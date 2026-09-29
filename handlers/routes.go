package handlers

import (
	"net/http"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"gorm.io/gorm"
)

// AppDependencies holds all handlers, repositories, and configuration needed for route registration.
type AppDependencies struct {
	Cfg       config.Config
	Hub       *realtime.Hub
	UserRepo  repository.UserRepository
	AlbumRepo repository.AlbumRepositoryInterface
	Store     *media.Store
	DB        *gorm.DB

	AlbumHandler           *AlbumHandler
	PersonHandler          *PersonHandler
	FaceHandler            *FaceHandler
	ImagePreviewHandler    *ImagePreviewHandler
	DebugHandler           *DebugHandler
	AuthHandler            *AuthHandler
	PermissionsHandler     *PermissionsHandler
	AdminUserHandler       *AdminUserHandler
	AdminRoleHandler       *AdminRoleHandler
	AdminInviteCodeHandler *AdminInviteCodeHandler
	AdminAlbumHandler      *AdminAlbumHandler
	AdminAlbumUserHandler  *AdminAlbumUserHandler
	AdminAlbumGroupHandler *AdminAlbumGroupHandler
	AlbumGroupHandler      *AlbumGroupHandler
	AdminImageTagHandler   *AdminImageTagHandler
	AdminCollectionHandler *AdminCollectionHandler
	CollectionHandler      *CollectionHandler
	SetupHandler           *SetupHandler
}

// RegisterRoutes registers all middleware and API routes on the given router.
func RegisterRoutes(r chi.Router, deps AppDependencies) {
	cfg := deps.Cfg

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS is scoped to only auth and admin routes to avoid Vary: Origin on public
	// cacheable endpoints (albums, groups, collections).
	credentialedCORS := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Content-Length"},
		ExposedHeaders:   []string{"Link"},
		MaxAge:           300,
		AllowCredentials: true,
	}).Handler

	r.Route("/api", func(r chi.Router) {
		r.Post("/setup/initial-admin", deps.SetupHandler.CreateFirstAdmin)

		// authentication routes
		r.Route("/auth", func(r chi.Router) {
			r.Use(credentialedCORS)
			r.Post("/login", deps.AuthHandler.Login)
			r.Post("/register", deps.AuthHandler.Register)
			r.Post("/logout", deps.AuthHandler.Logout)

			r.Group(func(r chi.Router) {
				r.Use(func(next http.Handler) http.Handler {
					return AuthMiddleware(deps.UserRepo, next)
				})
				r.Get("/me", deps.AuthHandler.CurrentUser)
				r.Put("/me", deps.AuthHandler.UpdateProfile)
			})
		})

		// permissions definition routes
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", deps.PermissionsHandler.ListDefinedPermissions)
			r.Get("/keys", deps.PermissionsHandler.ListDefinedPermissionKeys)
		})

		// admin routes for User and Role management
		r.Route("/admin", func(r chi.Router) {
			r.Use(credentialedCORS)
			r.Use(func(next http.Handler) http.Handler {
				return AuthMiddleware(deps.UserRepo, next)
			})

			// user management routes
			r.Route("/users", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("user.list", next)
				}).Get("/", deps.AdminUserHandler.ListUsers)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("user.create", next)
				}).Post("/", deps.AdminUserHandler.CreateUser)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("user.view", next)
					}).Get("/", deps.AdminUserHandler.GetUser)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("user.edit", next)
					}).Put("/", deps.AdminUserHandler.UpdateUser)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("user.delete", next)
					}).Delete("/", deps.AdminUserHandler.DeleteUser)
				})
			})

			// role management routes
			r.Route("/roles", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireAnyGlobalPermission([]string{"role.list", "role.view", "role.create", "role.edit", "role.delete"}, next)
				}).Get("/", deps.AdminRoleHandler.ListRoles)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("role.create", next)
				}).Post("/", deps.AdminRoleHandler.CreateRole)

				r.Route("/{roleID}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("role.view", next)
					}).Get("/", deps.AdminRoleHandler.GetRole)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("role.edit", next)
					}).Put("/", deps.AdminRoleHandler.UpdateRole)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("role.delete", next)
					}).Delete("/", deps.AdminRoleHandler.DeleteRole)

					// user-role association routes
					r.Route("/users", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return RequireGlobalPermission("role.view.users", next)
						}).Get("/", deps.AdminRoleHandler.GetRoleUsers)

						r.With(func(next http.Handler) http.Handler {
							return RequireGlobalPermission("role.edit.users", next)
						}).Post("/", deps.AdminRoleHandler.AddUserToRole)

						r.With(func(next http.Handler) http.Handler {
							return RequireGlobalPermission("role.edit.users", next)
						}).Delete("/{userID}", deps.AdminRoleHandler.RemoveUserFromRole)
					})
				})
			})

			// invite code management routes
			r.Route("/invite-codes", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireAnyGlobalPermission([]string{"invite.list", "invite.view", "invite.create", "invite.edit", "invite.delete"}, next)
				}).Get("/", deps.AdminInviteCodeHandler.ListInviteCodes)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("invite.create", next)
				}).Post("/", deps.AdminInviteCodeHandler.CreateInviteCode)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("invite.view", next)
					}).Get("/", deps.AdminInviteCodeHandler.GetInviteCode)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("invite.edit", next)
					}).Put("/", deps.AdminInviteCodeHandler.UpdateInviteCode)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("invite.delete", next)
					}).Delete("/", deps.AdminInviteCodeHandler.DeleteInviteCode)
				})
			})

			// album management routes
			r.Route("/albums", func(r chi.Router) {
				r.With(
					RequireAnyGlobalPermissionOrAlbumAccess([]string{"album.list", "album.view", "album.create", "album.edit.general", "album.delete"}),
				).Get("/", deps.AdminAlbumHandler.ListAlbums)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.create", next)
				}).Post("/", deps.AdminAlbumHandler.CreateAlbum)

				r.Route("/{id}", func(r chi.Router) {
					r.Use(ResolveAlbumSlug(deps.AlbumRepo))

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.list", "album.view.content", next)
					}).Get("/", deps.AdminAlbumHandler.GetAlbum)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.edit.general", next)
					}).Put("/", deps.AdminAlbumHandler.UpdateAlbum)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.delete", next)
					}).Delete("/", deps.AdminAlbumHandler.DeleteAlbum)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.editmeta", next)
					}).Post("/banners", deps.AdminAlbumHandler.AddAlbumBanner)

					r.Route("/banners/{bannerId}", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return RequireAlbumPermission("album.edit.general", "album.photo.editmeta", next)
						}).Delete("/", deps.AdminAlbumHandler.DeleteAlbumBanner)
					})

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.editmeta", next)
					}).Put("/banners/order", deps.AdminAlbumHandler.ReorderAlbumBanners)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.upload", next)
					}).Post("/upload", deps.AdminAlbumHandler.UploadImages)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.list", "album.view.content", next)
					}).Get("/images", deps.AdminAlbumHandler.ListAlbumImages)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.delete", next)
					}).Delete("/images", deps.AdminAlbumHandler.DeleteAlbumImage)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.view.content", next)
					}).Post("/zip", deps.AlbumHandler.RequestAlbumZipGeneration)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.list", "album.view.content", next)
					}).Get("/zip", deps.AlbumHandler.DownloadAlbumZipByID)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.list", "album.view.content", next)
					}).Get("/uploaders", deps.AdminAlbumHandler.GetAlbumUploaders)

					// album user management routes
					r.Route("/users", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return RequireAlbumPermission("album.manage.members.global", "album.manage.members", next)
						}).Get("/", deps.AdminAlbumUserHandler.GetAlbumUsers)

						r.With(func(next http.Handler) http.Handler {
							return RequireAlbumPermission("album.manage.members.global", "album.manage.members", next)
						}).Get("/available", deps.AdminAlbumUserHandler.GetAvailableUsers)

						r.With(func(next http.Handler) http.Handler {
							return RequireAlbumPermission("album.manage.members.global", "album.manage.members", next)
						}).Post("/", deps.AdminAlbumUserHandler.AddUserToAlbum)

						r.Route("/{userID}", func(r chi.Router) {
							r.With(func(next http.Handler) http.Handler {
								return RequireAlbumPermission("album.manage.members.global", "album.manage.members", next)
							}).Put("/", deps.AdminAlbumUserHandler.UpdateUserAlbumPermissions)

							r.With(func(next http.Handler) http.Handler {
								return RequireAlbumPermission("album.manage.members.global", "album.manage.members", next)
							}).Delete("/", deps.AdminAlbumUserHandler.RemoveUserFromAlbum)
						})
					})

					// album group assignment
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.group.manage", next)
					}).Put("/group", deps.AdminAlbumGroupHandler.SetAlbumGroup)

					// album default tags
					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.editmeta", next)
					}).Get("/default-tags", deps.AdminImageTagHandler.GetAlbumDefaultTags)

					r.With(func(next http.Handler) http.Handler {
						return RequireAlbumPermission("album.edit.general", "album.photo.editmeta", next)
					}).Put("/default-tags", deps.AdminImageTagHandler.SetAlbumDefaultTags)
				})
			})

			// album group management routes
			r.Route("/groups", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.group.manage", next)
				}).Get("/", deps.AdminAlbumGroupHandler.ListGroups)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.group.manage", next)
				}).Post("/", deps.AdminAlbumGroupHandler.CreateGroup)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.group.manage", next)
					}).Get("/", deps.AdminAlbumGroupHandler.GetGroup)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.group.manage", next)
					}).Put("/", deps.AdminAlbumGroupHandler.UpdateGroup)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.group.manage", next)
					}).Delete("/", deps.AdminAlbumGroupHandler.DeleteGroup)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("album.group.manage", next)
					}).Put("/banner", deps.AdminAlbumGroupHandler.UploadGroupBanner)
				})
			})

			// image tag management routes
			r.Route("/images", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.edit.general", next)
				}).Get("/tags", deps.AdminImageTagHandler.GetImageTags)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.edit.general", next)
				}).Post("/tags", deps.AdminImageTagHandler.AddManualTag)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("album.edit.general", next)
				}).Delete("/tags", deps.AdminImageTagHandler.RemoveManualTag)
			})

			// collection management routes
			r.Route("/collections", func(r chi.Router) {
				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("collection.manage", next)
				}).Get("/", deps.AdminCollectionHandler.ListCollections)

				r.With(func(next http.Handler) http.Handler {
					return RequireGlobalPermission("collection.manage", next)
				}).Post("/", deps.AdminCollectionHandler.CreateCollection)

				r.Route("/{id}", func(r chi.Router) {
					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Get("/", deps.AdminCollectionHandler.GetCollection)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Put("/", deps.AdminCollectionHandler.UpdateCollection)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Delete("/", deps.AdminCollectionHandler.DeleteCollection)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Post("/banners", deps.AdminCollectionHandler.AddCollectionBanner)

					r.Route("/banners/{bannerId}", func(r chi.Router) {
						r.With(func(next http.Handler) http.Handler {
							return RequireGlobalPermission("collection.manage", next)
						}).Delete("/", deps.AdminCollectionHandler.DeleteCollectionBanner)
					})

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Put("/banners/order", deps.AdminCollectionHandler.ReorderCollectionBanners)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Put("/banners/inherit", deps.AdminCollectionHandler.SetCollectionInheritBanners)

					r.With(func(next http.Handler) http.Handler {
						return RequireGlobalPermission("collection.manage", next)
					}).Put("/filters", deps.AdminCollectionHandler.SetCollectionFilters)
				})
			})
		})

		r.Route("/albums", func(r chi.Router) {
			r.Get("/", deps.AlbumHandler.ListAlbums)
			r.Route("/{album_identifier}", func(r chi.Router) {
				r.Get("/", deps.AlbumHandler.GetAlbum)
				r.Get("/contents", deps.AlbumHandler.GetAlbumContents)
				r.Get("/zip", deps.AlbumHandler.DownloadAlbumZip)
			})
		})

		r.Route("/groups", func(r chi.Router) {
			r.Get("/", deps.AlbumGroupHandler.ListGroups)
			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/", deps.AlbumGroupHandler.GetGroup)
				r.Get("/photos", deps.AlbumGroupHandler.GetGroupPhotos)
			})
		})

		r.Route("/collections", func(r chi.Router) {
			r.Get("/", deps.CollectionHandler.ListCollections)
			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/", deps.CollectionHandler.GetCollection)
				r.Get("/photos", deps.CollectionHandler.GetCollectionPhotos)
			})
		})

		r.Route("/share", func(r chi.Router) {
			r.Route("/albums", func(r chi.Router) {
				r.Get("/{album_identifier}", deps.AlbumHandler.ShareAlbumHTML)
			})
			r.Route("/collections", func(r chi.Router) {
				r.Get("/{slug}", deps.CollectionHandler.ShareCollectionHTML)
			})
		})

		// helper to require login + the "people.manage" global permission for mutating people/alias routes
		requirePeopleManage := func(next http.Handler) http.Handler {
			return AuthMiddleware(deps.UserRepo, RequireGlobalPermission("people.manage", next))
		}
		// helper to require login + the "face.manage" global permission for mutating face routes
		requireFaceManage := func(next http.Handler) http.Handler {
			return AuthMiddleware(deps.UserRepo, RequireGlobalPermission("face.manage", next))
		}

		r.Route("/people", func(r chi.Router) {
			r.With(credentialedCORS, requirePeopleManage).Post("/", deps.PersonHandler.CreatePerson)
			r.Get("/", deps.PersonHandler.ListPeople)
			r.Get("/search", deps.PersonHandler.SearchPeople)
			r.Route("/{person_id}", func(r chi.Router) {
				r.Get("/", deps.PersonHandler.GetPerson)
				r.With(credentialedCORS, requirePeopleManage).Put("/", deps.PersonHandler.UpdatePerson)
				r.With(credentialedCORS, requirePeopleManage).Delete("/", deps.PersonHandler.DeletePerson)
				r.With(credentialedCORS, requirePeopleManage).Put("/key-photo", deps.PersonHandler.SetKeyPhoto)
				r.Get("/key-photo.jpg", deps.PersonHandler.ServeKeyPhoto)
				r.Route("/aliases", func(r chi.Router) {
					r.Get("/", deps.PersonHandler.ListAliases)
					r.With(credentialedCORS, requirePeopleManage).Post("/", deps.PersonHandler.AddAlias)
					r.With(credentialedCORS, requirePeopleManage).Delete("/{alias_id}", deps.PersonHandler.DeleteAlias)
				})
			})
		})

		r.Route("/images/faces", func(r chi.Router) {
			r.With(credentialedCORS, requireFaceManage).Post("/", deps.FaceHandler.AddFace)
			r.Get("/", deps.FaceHandler.ListFacesByImage)
		})

		r.Route("/faces", func(r chi.Router) {
			r.With(credentialedCORS, requireFaceManage).Get("/untagged", deps.FaceHandler.GetUntaggedFaces)
			r.Route("/{face_id}", func(r chi.Router) {
				r.With(credentialedCORS, requireFaceManage).Get("/", deps.FaceHandler.GetFace)
				r.With(credentialedCORS, requireFaceManage).Put("/", deps.FaceHandler.UpdateFace)
				r.With(credentialedCORS, requireFaceManage).Delete("/", deps.FaceHandler.DeleteFace)
				r.With(credentialedCORS, requireFaceManage).Get("/similar", deps.FaceHandler.GetSimilarFaces)
				r.With(credentialedCORS, requireFaceManage).Get("/suggest", deps.FaceHandler.SuggestFace)
				r.With(credentialedCORS, requireFaceManage).Post("/tag", deps.FaceHandler.TagFace)
				r.With(credentialedCORS, requireFaceManage).Post("/auto-tag", deps.FaceHandler.AutoTagFace)
			})
		})

		r.Route("/search/faces", func(r chi.Router) {
			r.Get("/", deps.FaceHandler.SearchFacesByPerson)
		})

		// media served from object storage
		r.Get("/thumbnails/*", ObjectServer(deps.Store, media.PrefixThumbnails))
		r.Get("/album_banners/*", ObjectServer(deps.Store, media.PrefixBanners))
		r.Get("/originals/*", deps.ImagePreviewHandler.ServeOriginal)
		r.Get("/preview/*", deps.ImagePreviewHandler.ServeScaledPreview)

		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if err := deps.Store.Ping(r.Context()); err != nil {
				WriteAPIError(w, http.StatusServiceUnavailable, "ObjectStoreUnavailable", "object store unavailable")
				return
			}
			if sqlDB, err := deps.DB.DB(); err != nil || sqlDB.PingContext(r.Context()) != nil {
				WriteAPIError(w, http.StatusServiceUnavailable, "DatabaseUnavailable", "database unavailable")
				return
			}
			_, _ = w.Write([]byte("ok"))
		})

		r.Route("/debug", func(r chi.Router) {
			r.Use(credentialedCORS)
			r.Use(func(next http.Handler) http.Handler {
				return AuthMiddleware(deps.UserRepo, next)
			})
			r.Use(func(next http.Handler) http.Handler {
				return RequireGlobalPermission("system.debug.access", next)
			})

			r.Get("/image_with_faces", deps.ImagePreviewHandler.ServeImageWithFaces)
			r.Post("/queue_detection", deps.DebugHandler.QueueFaceDetection)
			r.Get("/detection_status", deps.DebugHandler.GetDetectionStatus)
			r.Get("/faces", deps.FaceHandler.DebugFaces)
		})
	})

	// websocket endpoint for realtime updates (authenticated).
	// The token is carried via the Sec-WebSocket-Protocol header (as "bearer, <token>")
	// rather than a query parameter so it never gets written to access logs by
	// middleware.Logger, which logs the full request URI.
	r.Get("/api/ws", func(w http.ResponseWriter, req *http.Request) {
		if protos := websocket.Subprotocols(req); len(protos) > 1 && protos[0] == "bearer" {
			req.Header.Set("Authorization", "Bearer "+protos[1])
		}
		AuthMiddleware(deps.UserRepo, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _ := r.Context().Value(UserContextKey).(*models.User)
			deps.Hub.ServeWS(w, r, user)
		})).ServeHTTP(w, req)
	})
}
