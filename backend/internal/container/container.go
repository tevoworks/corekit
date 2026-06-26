package container

import (
	"database/sql"
	"log"

	"github.com/tevoworks/corekit/backend/internal/authverify"
	"github.com/tevoworks/corekit/backend/internal/config"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/middleware"
	"github.com/tevoworks/corekit/backend/internal/modules/apikey"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
	"github.com/tevoworks/corekit/backend/internal/modules/cms"
	"github.com/tevoworks/corekit/backend/internal/modules/contact"
	"github.com/tevoworks/corekit/backend/internal/modules/iam"
	"github.com/tevoworks/corekit/backend/internal/modules/permregistry"
	"github.com/tevoworks/corekit/backend/internal/modules/queue"
	"github.com/tevoworks/corekit/backend/internal/modules/rbac"
	"github.com/tevoworks/corekit/backend/internal/modules/settings"
	storagepkg "github.com/tevoworks/corekit/backend/internal/modules/storage"
	"github.com/tevoworks/corekit/backend/internal/modules/webhook"
	"github.com/tevoworks/corekit/backend/internal/redisstore"
	"github.com/tevoworks/corekit/backend/pkg/event"
)

type Container struct {
	DB       *sql.DB
	RevStore *redisstore.RevocationStore

	AuditSvc      audit.Service
	RBACSvc       rbac.Service
	AuthVerifySvc authverify.Service
	IAMSvc        iam.Service
	SettingsSvc   settings.Service
	StorageSvc    storagepkg.Service
	CMSSvc        cms.Service
	ContactSvc    contact.Service
	APIKeySvc     apikey.Service
	WebhookSvc    webhook.Service
	PermRegSvc    permregistry.Service

	AuditH      *audit.Handler
	QueueH      *queue.Handler
	RBACH       *rbac.Handler
	IAMH        *iam.Handler
	SettingsH   *settings.Handler
	StorageH    *storagepkg.Handler
	CMSH        *cms.Handler
	ContactH    *contact.Handler
	APIKeyH     *apikey.Handler
	WebhookH    *webhook.Handler
	PermRegH    *permregistry.Handler
	AuthVerifyH *authverify.Handler

	EventDispatcher *event.EventDispatcher
	QueueRepo       queue.Repository
	IntroCache      *authverify.IntrospectionCache
	Config          *config.Config
}

func NewContainer(cfg *config.Config) *Container {
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.RunMigrations(db, "migrations/"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	middleware.InitRateLimiters(cfg.RedisURL, cfg.AppEnv)

	queueRepo := queue.NewRepository(db)
	eventDispatcher := event.NewEventDispatcher(queueRepo)

	revStore := redisstore.NewRevocationStore(cfg.RedisURL)
	cache := authverify.NewIntrospectionCache()

	auditRepo := audit.NewRepository(db)
	auditSvc := audit.NewService(auditRepo)
	auditH := audit.NewHandler(auditSvc)

	rbacRepo := rbac.NewRepository(db)
	rbacSvc := rbac.NewService(db, rbacRepo, auditSvc)
	rbacH := rbac.NewHandler(rbacSvc)

	authVerifySvc := authverify.NewService(db, cfg.JWTSecret, cache, revStore)
	authVerifyH := authverify.NewHandler(authVerifySvc)

	settingsRepo := settings.NewRepository(db)
	settingsSvc := settings.NewService(db, settingsRepo, auditSvc)
	settingsH := settings.NewHandler(settingsSvc, rbacSvc)

	cmsRepo := cms.NewRepository(db)
	cmsSvc := cms.NewService(db, cmsRepo, auditSvc)
	cmsH := cms.NewHandler(cmsSvc, rbacSvc)

	contactRepo := contact.NewRepository(db)
	contactSvc := contact.NewService(db, contactRepo, auditSvc)
	contactH := contact.NewHandler(contactSvc, rbacSvc)

	iamRepo := iam.NewRepository(db)
	iamSvc := iam.NewService(db, iamRepo, cfg.JWTSecret, auditSvc, revStore, queueRepo, eventDispatcher, cache, cfg.FrontendURL)
	iamH := iam.NewHandler(iamSvc, rbacSvc, settingsSvc, db, cfg.JWTSecret, cfg.AppEnv, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.FrontendURL, cfg.RedisURL)

	storageRepo := storagepkg.NewRepository(db)
	storageProvider, err := storagepkg.NewS3StorageProvider(cfg.S3Endpoint, cfg.S3Region, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		log.Fatalf("Failed to create S3 storage provider: %v", err)
	}
	storageSvc := storagepkg.NewService(db, storageRepo, storageProvider, auditSvc)
	storageH := storagepkg.NewHandler(storageSvc, rbacSvc, settingsSvc)

	apikeyRepo := apikey.NewRepository(db)
	apikeySvc := apikey.NewService(db, apikeyRepo, auditSvc)
	apikeyH := apikey.NewHandler(apikeySvc, rbacSvc)

	webhookRepo := webhook.NewRepository(db)
	webhookSvc := webhook.NewService(db, webhookRepo, queueRepo, auditSvc, cfg.WebhookEncryptionKey)
	webhookH := webhook.NewHandler(webhookSvc, rbacSvc)

	permRegRepo := permregistry.NewRepository(db)
	permRegSvc := permregistry.NewService(db, permRegRepo, auditSvc)
	permRegH := permregistry.NewHandler(permRegSvc)

	queueH := queue.NewHandler(queueRepo)

	return &Container{
		DB:              db,
		RevStore:        revStore,
		AuditSvc:        auditSvc,
		RBACSvc:         rbacSvc,
		AuthVerifySvc:   authVerifySvc,
		IAMSvc:          iamSvc,
		SettingsSvc:     settingsSvc,
		StorageSvc:      storageSvc,
		CMSSvc:          cmsSvc,
		ContactSvc:      contactSvc,
		APIKeySvc:       apikeySvc,
		WebhookSvc:      webhookSvc,
		PermRegSvc:      permRegSvc,
		AuditH:          auditH,
		QueueH:          queueH,
		RBACH:           rbacH,
		IAMH:            iamH,
		SettingsH:       settingsH,
		StorageH:        storageH,
		CMSH:            cmsH,
		ContactH:        contactH,
		APIKeyH:         apikeyH,
		WebhookH:        webhookH,
		PermRegH:        permRegH,
		AuthVerifyH:     authVerifyH,
		EventDispatcher: eventDispatcher,
		QueueRepo:       queueRepo,
		IntroCache:      cache,
		Config:          cfg,
	}
}
