package test

import (
	"context"
	"fmt"
	"strings"
	"time"
	"database/sql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/stretchr/testify/require"
	"github.com/AdamKorcz/go-118-fuzz-build/testing"
	"github.com/usememos/memos/store"
	"github.com/usememos/memos/store/db"
	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/internal/version"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func FuzzTestMemoOperations(f *testing.F) {

	testCases := []struct {
		content1	string
		content2	string
	}{{"a", "b"}}

	for _, tc := range testCases {
		f.Add(tc.content1, tc.content2)
	}

	f.Fuzz(func(t *testing.T, content1 string, content2 string) {
		ctx := context.Background()
		ts := NewTestingStoreFuzz(ctx, t)

		user, err := createTestingHostUser(ctx, ts)
		require.NoError(t, err)
		visibility := store.Public
		user_id := "const"
		memo1 := &store.Memo{
			UID:		user_id,
			CreatorID:	user.ID,
			Content:	content1,
			Visibility:	visibility,
		}

		createdMemo, err := ts.CreateMemo(ctx, memo1)
		require.NoError(t, err)
		require.NotNil(t, createdMemo)
		require.NotZero(t, createdMemo.ID)
		require.Equal(t, memo1.Content, createdMemo.Content)
		require.Equal(t, memo1.Visibility, createdMemo.Visibility)
		user_id2 := "skibi"
		memo2 := &store.Memo{
			UID:		user_id2,
			CreatorID:	user.ID,
			Content:	content2,
			Visibility:	store.Private,
		}

		createdMemo2, err := ts.CreateMemo(ctx, memo2)
		require.NoError(t, err)
		require.NotNil(t, createdMemo2)
		require.NotZero(t, createdMemo2.ID)
		require.Equal(t, memo2.Content, createdMemo2.Content)
		require.Equal(t, memo2.Visibility, createdMemo2.Visibility)

		newContent := content1 + " updated"
		update := &store.UpdateMemo{
			ID:		createdMemo.ID,
			Content:	&newContent,
			Visibility:	&visibility,
		}

		err = ts.UpdateMemo(ctx, update)
		require.NoError(t, err)

		get := &store.FindMemo{
			ID: &createdMemo.ID,
		}
		updatedMemo, err := ts.GetMemo(ctx, get)
		require.NoError(t, err)
		require.NotNil(t, updatedMemo)
		require.Equal(t, newContent, updatedMemo.Content)

		deleteMemo := &store.DeleteMemo{
			ID: createdMemo.ID,
		}

		err = ts.DeleteMemo(ctx, deleteMemo)
		require.NoError(t, err)
		deletedMemo, err := ts.GetMemo(ctx, get)
		require.NoError(t, err)
		require.Nil(t, deletedMemo)

	})
}

func createTestingHostUser(ctx context.Context, ts *store.Store) (*store.User, error) {
	userCreate := &store.User{
		Username:	"test",
		Role:		store.RoleHost,
		Email:		"test@test.com",
		Nickname:	"test_nickname",
		Description:	"test_description",
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	userCreate.PasswordHash = string(passwordHash)
	user, err := ts.CreateUser(ctx, userCreate)
	return user, err
}

func NewTestingStoreFuzz(ctx context.Context, t *testing.T) *store.Store {
	driver := getDriverFromEnv()
	profile := getTestingProfileForDriverFuzz(t, driver)
	dbDriver, err := db.NewDBDriver(profile)
	if err != nil {
		t.Fatalf("failed to create db driver: %v", err)
	}

	store := store.New(dbDriver, profile)
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	return store
}

func getTestingProfileForDriverFuzz(t *testing.T, driver string) *profile.Profile {

	_ = godotenv.Load(".env")

	dir := t.TempDir()
	mode := "prod"
	port := getUnusedPort()

	var dsn string
	switch driver {
	case "sqlite":
		dsn = fmt.Sprintf("%s/memos_%s.db", dir, mode)
	case "mysql":
		dsn = GetMySQLDSNFuzz(t)
	case "postgres":
		dsn = GetPostgresDSNFuzz(t)
	default:
		t.Fatalf("unsupported driver: %s", driver)
	}

	return &profile.Profile{
		Mode:		mode,
		Port:		port,
		Data:		dir,
		DSN:		dsn,
		Driver:		driver,
		Version:	version.GetCurrentVersion(mode),
	}
}

func GetMySQLDSNFuzz(t *testing.T) string {
	ctx := context.Background()

	mysqlOnce.Do(func() {
		nw, err := getTestNetwork(ctx)
		if err != nil {
			t.Fatalf("failed to create test network: %v", err)
		}

		container, err := mysql.Run(ctx,
			"mysql:8",
			mysql.WithDatabase("init_db"),
			mysql.WithUsername("root"),
			mysql.WithPassword(testPassword),
			testcontainers.WithEnv(map[string]string{
				"MYSQL_ROOT_PASSWORD": testPassword,
			}),
			testcontainers.WithWaitStrategy(
				wait.ForAll(
					wait.ForLog("ready for connections").WithOccurrence(2),
					wait.ForListeningPort("3306/tcp"),
				).WithDeadline(120*time.Second),
			),
			network.WithNetwork(nil, nw),
		)
		if err != nil {
			t.Fatalf("failed to start MySQL container: %v", err)
		}
		mysqlContainer.Store(container)

		dsn, err := container.ConnectionString(ctx, "multiStatements=true")
		if err != nil {
			t.Fatalf("failed to get MySQL connection string: %v", err)
		}

		if err := waitForDB("mysql", dsn, 30*time.Second); err != nil {
			t.Fatalf("MySQL not ready for connections: %v", err)
		}

		mysqlBaseDSN.Store(dsn)
	})

	dsn, ok := mysqlBaseDSN.Load().(string)
	if !ok || dsn == "" {
		t.Fatal("MySQL container failed to start in a previous test")
	}

	dbCreationMutex.Lock()
	defer dbCreationMutex.Unlock()

	dbName := fmt.Sprintf("memos_test_%d", dbCounter.Add(1))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName)); err != nil {
		t.Fatalf("failed to create database %s: %v", dbName, err)
	}

	return strings.Replace(dsn, "/init_db?", "/"+dbName+"?", 1)
}

func GetPostgresDSNFuzz(t *testing.T) string {
	ctx := context.Background()

	postgresOnce.Do(func() {
		nw, err := getTestNetwork(ctx)
		if err != nil {
			t.Fatalf("failed to create test network: %v", err)
		}

		container, err := postgres.Run(ctx,
			"postgres:18",
			postgres.WithDatabase("init_db"),
			postgres.WithUsername(testUser),
			postgres.WithPassword(testPassword),
			testcontainers.WithWaitStrategy(
				wait.ForAll(
					wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
					wait.ForListeningPort("5432/tcp"),
				).WithDeadline(120*time.Second),
			),
			network.WithNetwork(nil, nw),
		)
		if err != nil {
			t.Fatalf("failed to start PostgreSQL container: %v", err)
		}
		postgresContainer.Store(container)

		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("failed to get PostgreSQL connection string: %v", err)
		}

		if err := waitForDB("postgres", dsn, 30*time.Second); err != nil {
			t.Fatalf("PostgreSQL not ready for connections: %v", err)
		}

		postgresBaseDSN.Store(dsn)
	})

	dsn, ok := postgresBaseDSN.Load().(string)
	if !ok || dsn == "" {
		t.Fatal("PostgreSQL container failed to start in a previous test")
	}

	dbCreationMutex.Lock()
	defer dbCreationMutex.Unlock()

	dbName := fmt.Sprintf("memos_test_%d", dbCounter.Add(1))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		t.Fatalf("failed to create database %s: %v", dbName, err)
	}

	return strings.Replace(dsn, "/init_db?", "/"+dbName+"?", 1)
}
