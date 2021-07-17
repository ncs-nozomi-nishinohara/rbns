package restserver

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/n-creativesystem/rbns/domain/repository"
	"github.com/n-creativesystem/rbns/logger"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	pHandler   permissionHandle
	rHandler   roleHandle
	oHandler   organizationHandle
	uHandler   userHandle
	keyHandler apiKeyHandle
)

type webUI struct {
	enabled bool
	prefix  string
	root    string
	indexes bool
}

type Option func(conf *config)

type config struct {
	secure    bool
	whiteList []*net.IPNet
	ui        webUI
	debug     bool
}

func WithDebug(conf *config) {
	conf.debug = true
}

func WithSecure(repo repository.ApiKey) Option {
	return func(conf *config) {
		conf.secure = true
		keyHandler = newApiKeyHander(repo)
	}
}

func WithWhiteList(whitelistIp string) Option {
	ips := strings.Split(whitelistIp, ",")
	ipNets := make([]*net.IPNet, len(ips))
	for idx, ip := range ips {
		_, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			panic(err)
		}
		ipNets[idx] = ipNet
	}
	return func(conf *config) {
		copy(conf.whiteList, ipNets)
	}
}

func WithUI(enabled bool, prefix, root string, indexes bool) Option {
	return func(conf *config) {
		conf.ui = webUI{
			enabled: enabled,
			prefix:  prefix,
			root:    root,
			indexes: indexes,
		}
	}
}

// New is *gin.Engine
//
// Endpoints base is `/api/v1/`
//
// • /permissions
//
// • /roles
//
// • /organizations
//
// • /users
func New(con *grpc.ClientConn, opts ...Option) *gin.Engine {
	conf := &config{}
	for _, opt := range opts {
		opt(conf)
	}
	loggerOpts := []logger.HandlerLogOption{}
	log := logger.NewHandlerLogger()
	if conf.debug {
		log.SetLevel(logrus.DebugLevel)
		loggerOpts = append(loggerOpts, logger.WithGinDebug(logrus.DebugLevel))
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
		log.SetLevel(logrus.InfoLevel)
	}
	gin.DefaultWriter = log
	r := gin.New()
	r.Use(logger.RestLogger(loggerOpts...), gin.Recovery())
	if conf.ui.enabled {
		fileSystem := newFileSystem(conf.ui.root, conf.ui.indexes)
		r.Use(staticServe(conf.ui.prefix, fileSystem))
	}
	if len(conf.whiteList) > 0 {
		r.Use(ipFilter(conf.whiteList))
	}
	if conf.secure {
		r.Use(setApiKey)
		r.GET("/apikey", keyHandler.get)
		r.PUT("/apikey", keyHandler.generate)
	}
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  http.StatusNotFound,
			"message": http.StatusText(http.StatusNotFound),
		})
	})
	r.Use(errorMiddleware)
	apiV1 := r.Group("/api/v1")
	pHandler = newPermissionHandler(con)
	rHandler = newRoleHandler(con)
	oHandler = newOrganizationHandler(con)
	uHandler = newUserHandler(con)
	p := apiV1.Group("/permissions")
	{
		p.GET("", pHandler.findAll)
		p.POST("", pHandler.create)
		p.GET("/:id", pHandler.findById)
		p.PUT("/:id", pHandler.update)
		p.DELETE("/:id", pHandler.delete)
	}
	role := apiV1.Group("/roles")
	{
		role.GET("", rHandler.findAll)
		role.POST("", rHandler.create)
		role.GET("/:id", rHandler.findById)
		role.PUT("/:id", rHandler.update)
		role.DELETE("/:id", rHandler.delete)
		role.GET("/:id/permissions", rHandler.getPermissions)
		role.PUT("/:id/permissions", rHandler.addPermissions)
		role.DELETE("/:id/permissions/:permissionId", rHandler.deletePermissions)
	}
	organization := apiV1.Group("/organizations")
	{
		organization.GET("", oHandler.findAll)
		organization.POST("", oHandler.create)
		organization.GET("/:id", oHandler.findById)
		organization.PUT("/:id", oHandler.update)
		organization.DELETE("/:id", oHandler.delete)
		organization.POST("/:id/users", uHandler.create)
		organization.GET("/:id/users/:key", uHandler.findByKey)
		organization.DELETE("/:id/users/:key", uHandler.delete)
		organization.PUT("/:id/users/:key/roles", uHandler.addRole)
		organization.DELETE("/:id/users/:key/roles/:roleId", uHandler.deleteRole)
	}
	return r
}
