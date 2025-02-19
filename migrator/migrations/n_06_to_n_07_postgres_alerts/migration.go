// Code originally generated by pg-bindings generator.

package n6ton7

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	legacy "github.com/stackrox/rox/migrator/migrations/n_06_to_n_07_postgres_alerts/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_06_to_n_07_postgres_alerts/postgres"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"gorm.io/gorm"
)

var (
	startingSeqNum = pkgMigrations.BasePostgresDBVersionSeqNum() + 6 // 117

	migration = types.Migration{
		StartingSeqNum: startingSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startingSeqNum + 1)}, // 118
		Run: func(databases *types.Databases) error {
			legacyStore, err := legacy.New(databases.PkgRocksDB)
			if err != nil {
				return err
			}
			if err := move(databases.DBCtx, databases.GormDB, databases.PostgresDB, legacyStore); err != nil {
				return errors.Wrap(err,
					"moving alerts from rocksdb to postgres")
			}
			return nil
		},
	}
	batchSize = 10000
	log       = loghelper.LogWrapper{}
)

func move(ctx context.Context, gormDB *gorm.DB, postgresDB postgres.DB, legacyStore legacy.Store) error {
	store := pgStore.New(postgresDB)
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableAlertsStmt)

	var alerts []*storage.Alert
	err := walk(ctx, legacyStore, func(obj *storage.Alert) error {
		alerts = append(alerts, obj)
		if len(alerts) == batchSize {
			if err := store.UpsertMany(ctx, alerts); err != nil {
				log.WriteToStderrf("failed to persist alerts to store %v", err)
				return err
			}
			alerts = alerts[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(alerts) > 0 {
		if err = store.UpsertMany(ctx, alerts); err != nil {
			log.WriteToStderrf("failed to persist alerts to store %v", err)
			return err
		}
	}
	return nil
}

func walk(ctx context.Context, s legacy.Store, fn func(obj *storage.Alert) error) error {
	return s.Walk(ctx, fn)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
