package main

import (
	"fmt"
	"github.com/camden-git/mediasysbackend/media"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Info: No .env file found or error loading: %v", err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	storagePaths := []string{cfg.ThumbnailsPath, cfg.BannersPath, cfg.ArchivesPath, filepath.Dir(cfg.DatabasePath)}
	for _, p := range storagePaths {
		log.Printf("Ensuring storage directory exists: %s", p)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("FATAL: Failed to create storage directory %s: %v", p, err)
		}
	}

	gormDB, err := database.InitGormDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize GORM database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get underlying sql.DB from GORM: %v", err)
	}
	defer sqlDB.Close()

	log.Println("Running GORM AutoMigrate...")
	if err := database.AutoMigrateModels(gormDB); err != nil {
		log.Fatalf("FATAL: Failed to auto-migrate GORM models: %v", err)
	}
	log.Println("GORM AutoMigrate completed.")

	mediaSubDirs := map[media.AssetType]string{
		media.AssetTypeThumbnail: filepath.Base(cfg.ThumbnailsPath),
		media.AssetTypeBanner:    filepath.Base(cfg.BannersPath),
		media.AssetTypeArchive:   filepath.Base(cfg.ArchivesPath),
	}
	mediaStore, err := media.NewLocalStorage(cfg.MediaStoragePath, mediaSubDirs)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize media store: %v", err)
	}
	mediaProcessor := media.NewProcessor(mediaStore)

	log.Printf("Initializing image processor worker pool (Workers: %d, Queue Size: %d)...", cfg.NumThumbnailWorkers, cfg.ThumbnailQueueSize)

	albumRepo := repository.NewAlbumRepository(gormDB)
	personRepo := repository.NewPersonRepository(gormDB)
	faceRepo := repository.NewFaceRepository(gormDB)
	imageRepo := repository.NewImageRepository(gormDB)
	userRepo := repository.NewGormUserRepository(gormDB)
	roleRepo := repository.NewGormRoleRepository(gormDB)
	inviteCodeRepo := repository.NewGormInviteCodeRepository(gormDB)

	imageProcessor := workers.NewImageProcessor(
		cfg,
		imageRepo,
		albumRepo,
		faceRepo,
		cfg.ThumbnailQueueSize,
		cfg.NumThumbnailWorkers,
	)

	log.Printf("Serving files from root: %s", cfg.RootDirectory)
	log.Printf("Using database: %s", cfg.DatabasePath)
	log.Printf("Storing thumbnails in: %s", cfg.ThumbnailsPath)
	log.Printf("Thumbnail max size (longest side): %dpx", cfg.ThumbnailMaxSize)

	r := chi.NewRouter()

	corsOptions := cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Content-Length"},
		ExposedHeaders:   []string{"Link"},
		MaxAge:           300,
		AllowCredentials: true,
	}

	corsHandler := cors.New(corsOptions)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsHandler.Handler)

	albumHandler := &handlers.AlbumHandler{AlbumRepo: albumRepo, ImageRepo: imageRepo, Cfg: cfg, ThumbGen: imageProcessor, MediaProcessor: mediaProcessor}
	personHandler := &handlers.PersonHandler{PersonRepo: personRepo}
	faceHandler := &handlers.FaceHandler{FaceRepo: faceRepo, PersonRepo: personRepo, Cfg: cfg}
	imagePreviewHandler := &handlers.ImagePreviewHandler{FaceRepo: faceRepo, Cfg: cfg}
	authHandler := handlers.NewAuthHandler(userRepo, inviteCodeRepo) // Pass inviteCodeRepo
	permissionsHandler := handlers.NewPermissionsHandler()
	adminUserHandler := handlers.NewAdminUserHandler(userRepo, roleRepo)
	adminRoleHandler := handlers.NewAdminRoleHandler(roleRepo)
	adminInviteCodeHandler := handlers.NewAdminInviteCodeHandler(inviteCodeRepo)
	adminAlbumHandler := handlers.NewAdminAlbumHandler(albumRepo, imageRepo, userRepo, roleRepo, cfg)
	adminAlbumUserHandler := handlers.NewAdminAlbumUserHandler(userRepo, albumRepo)
	setupHandler := handlers.NewSetupHandler(gormDB, userRepo, roleRepo) // Initialize SetupHandler

	if err := handlers.SyncSuperAdminRole(roleRepo); err != nil {
		log.Fatalf("Failed to sync super admin role: %v", err)
	}

	r.Route("/api", func(r chi.Router) {
		r.Post("/setup/initial-admin", setupHandler.CreateFirstAdmin)

		// authentication routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/register", authHandler.Register)
			r.Post("/logout", authHandler.Logout)

			r.Group(func(r chi.Router) {
				r.Use(func(next http.Handler) http.Handler {
					return handlers.AuthMiddleware(userRepo, next)
				})
				r.Get("/me", authHandler.CurrentUser)
			})
		})

		// permissions definition routes
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", permissionsHandler.ListDefinedPermissions)
			r.Get("/keys", permissionsHandler.ListDefinedPermissionKeys)
		})

		// admin routes for User and Role management
		r.Route("/admin", func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return handlers.AuthMiddleware(userRepo, next) // All admin routes require authentication
			})

			// user management Routes
			r.Route("/users", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireGlobalPermission("user.list", next)
				}).Get("/", adminUserHandler.ListUsers)

				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireGlobalPermission("user.create", next)
				}).Post("/", adminUserHandler.CreateUser)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("user.view", next)
					}).Get("/", adminUserHandler.GetUser)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("user.edit", next)
					}).Put("/", adminUserHandler.UpdateUser)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("user.delete", next)
					}).Delete("/", adminUserHandler.DeleteUser)
				})
			})

			// role management Routes
			r.Route("/roles", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireAnyGlobalPermission([]string{"role.list", "role.view", "role.create", "role.edit", "role.delete"}, next)
				}).Get("/", adminRoleHandler.ListRoles)

				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireGlobalPermission("role.create", next)
				}).Post("/", adminRoleHandler.CreateRole)

				r.Route("/{roleID}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("role.view", next)
					}).Get("/", adminRoleHandler.GetRole)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("role.edit", next)
					}).Put("/", adminRoleHandler.UpdateRole)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("role.delete", next)
					}).Delete("/", adminRoleHandler.DeleteRole)

					// user-role association routes
					r.Route("/users", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("role.view.users", next)
						}).Get("/", adminRoleHandler.GetRoleUsers)

						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("role.edit.users", next)
						}).Post("/", adminRoleHandler.AddUserToRole)

						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("role.edit.users", next)
						}).Delete("/{userID}", adminRoleHandler.RemoveUserFromRole)
					})
				})
			})

			// invite code management routes
			r.Route("/invite-codes", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireAnyGlobalPermission([]string{"invite.list", "invite.view", "invite.create", "invite.edit", "invite.delete"}, next)
				}).Get("/", adminInviteCodeHandler.ListInviteCodes)

				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireGlobalPermission("invite.create", next)
				}).Post("/", adminInviteCodeHandler.CreateInviteCode)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("invite.view", next)
					}).Get("/", adminInviteCodeHandler.GetInviteCode)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("invite.edit", next)
					}).Put("/", adminInviteCodeHandler.UpdateInviteCode)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("invite.delete", next)
					}).Delete("/", adminInviteCodeHandler.DeleteInviteCode)
				})
			})

			// album management routes
			r.Route("/albums", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireAnyGlobalPermission([]string{"album.list", "album.view", "album.create", "album.edit.general", "album.delete"}, next)
				}).Get("/", adminAlbumHandler.ListAlbums)

				r.With(func(next http.Handler) http.Handler {
					return handlers.RequireGlobalPermission("album.create", next)
				}).Post("/", adminAlbumHandler.CreateAlbum)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.list", next)
					}).Get("/", adminAlbumHandler.GetAlbum)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.edit.general", next)
					}).Put("/", adminAlbumHandler.UpdateAlbum)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.delete", next)
					}).Delete("/", adminAlbumHandler.DeleteAlbum)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.edit.general", next)
					}).Put("/banner", albumHandler.UploadAlbumBanner)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.edit.general", next)
					}).Post("/zip", albumHandler.RequestAlbumZipGeneration)

					r.With(func(next http.Handler) http.Handler {
						return handlers.RequireGlobalPermission("album.list", next)
					}).Get("/zip", albumHandler.DownloadAlbumZipByID)

					// Album user management routes
					r.Route("/users", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("album.manage.members.global", next)
						}).Get("/", adminAlbumUserHandler.GetAlbumUsers)

						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("album.manage.members.global", next)
						}).Get("/available", adminAlbumUserHandler.GetAvailableUsers)

						r.With(func(next http.Handler) http.Handler {
							return handlers.RequireGlobalPermission("album.manage.members.global", next)
						}).Post("/", adminAlbumUserHandler.AddUserToAlbum)

						r.Route("/{userID}", func(r chi.Router) {
							r.With(func(next http.Handler) http.Handler {
								return handlers.RequireGlobalPermission("album.manage.members.global", next)
							}).Put("/", adminAlbumUserHandler.UpdateUserAlbumPermissions)

							r.With(func(next http.Handler) http.Handler {
								return handlers.RequireGlobalPermission("album.manage.members.global", next)
							}).Delete("/", adminAlbumUserHandler.RemoveUserFromAlbum)
						})
					})
				})
			})
		})

		r.Route("/albums", func(r chi.Router) {
			r.Get("/", albumHandler.ListAlbums)
			r.Route("/{album_identifier}", func(r chi.Router) {
				r.Get("/", albumHandler.GetAlbum)
				r.Get("/contents", albumHandler.GetAlbumContents)
				r.Get("/zip", albumHandler.DownloadAlbumZip)
			})
		})

		r.Route("/people", func(r chi.Router) {
			r.Post("/", personHandler.CreatePerson)
			r.Get("/", personHandler.ListPeople)
			r.Route("/{person_id}", func(r chi.Router) {
				r.Get("/", personHandler.GetPerson)
				r.Put("/", personHandler.UpdatePerson)
				r.Delete("/", personHandler.DeletePerson)
				r.Route("/aliases", func(r chi.Router) {
					r.Post("/", personHandler.AddAlias)
					r.Delete("/{alias_id}", personHandler.DeleteAlias)
				})
			})
		})

		r.Route("/images/faces", func(r chi.Router) {
			r.Post("/", faceHandler.AddFace)
			r.Get("/", faceHandler.ListFacesByImage)
		})

		r.Route("/faces", func(r chi.Router) {
			r.Route("/{face_id}", func(r chi.Router) {
				r.Get("/", faceHandler.GetFace)
				r.Put("/", faceHandler.UpdateFace)
				r.Delete("/", faceHandler.DeleteFace)
			})
		})

		r.Route("/search/faces", func(r chi.Router) {
			r.Get("/", faceHandler.SearchFacesByPerson)
		})

		thumbnailSubDir := filepath.Base(cfg.ThumbnailsPath)
		r.Get(fmt.Sprintf("/%s/*", thumbnailSubDir), handlers.AssetServer(cfg.MediaStoragePath, thumbnailSubDir))
		log.Printf("Registered thumbnail server at /%s/*", thumbnailSubDir)

		bannerSubDir := filepath.Base(cfg.BannersPath)
		r.Get(fmt.Sprintf("/%s/*", bannerSubDir), handlers.AssetServer(cfg.MediaStoragePath, bannerSubDir))
		log.Printf("Registered banner server at /%s/*", bannerSubDir)

		archiveSubDir := filepath.Base(cfg.ArchivesPath)
		r.Get(fmt.Sprintf("/%s/*", archiveSubDir), handlers.AssetServer(cfg.MediaStoragePath, archiveSubDir))
		log.Printf("Registered archive server at /%s/*", archiveSubDir)

		r.Route("/debug", func(r chi.Router) {
			// GET /debug/image_with_faces?path=relative/path/to/image.jpg
			r.Get("/image_with_faces", imagePreviewHandler.ServeImageWithFaces)
		})

		r.Get("/*", handlers.DirectoryHandler(cfg, imageRepo, imageProcessor))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	serverAddr := ":" + port
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Server listening on %s", serverAddr)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
