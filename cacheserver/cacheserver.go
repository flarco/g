package cacheserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/allegro/bigcache/v2"
	"github.com/flarco/gutil"
)

const (
	// base HTTP paths.
	apiVersion  = "v1"
	apiBasePath = "/api/" + apiVersion + "/"

	// path to cache.
	cachePath = apiBasePath + "cache/"
	statsPath = apiBasePath + "stats"

	// server version.
	version = "1.0.0"
)

var (
	port    int
	logfile string
	ver     bool

	// cache-specific settings.
	cache  *bigcache.BigCache
	config = bigcache.Config{
		Shards:             1024,
		MaxEntriesInWindow: 1000 * 10 * 60,
		LifeWindow:         100000 * 100000 * 60,
		HardMaxCacheSize:   8192,
		MaxEntrySize:       500,
	}
)

// CacheServer is a bigcache server
type CacheServer struct {
	Context gutil.Context
	Port    int
	logger  *log.Logger
}

// NewCacheServer creats a new instance of a cache server
func NewCacheServer(port int) *CacheServer {
	cacheServer := CacheServer{
		Context: gutil.NewContext(context.Background()),
		Port:    port,
	}
	return &cacheServer
}

// URL the cache server URL
func (cs *CacheServer) URL() string {
	hostname, _ := os.Hostname()
	u := gutil.F("http://%s:%d/api/v1/cache/", hostname, cs.Port)
	return u
}

// SetLogger sets the logger
func (cs *CacheServer) SetLogger() {

	var logger *log.Logger

	if logfile == "" {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		logger = log.New(f, "", log.LstdFlags)
	}

	var err error
	cache, err = bigcache.NewBigCache(config)
	if err != nil {
		logger.Fatal(err)
	}

	cs.logger = logger

	cs.logger.Print("cache initialised.")

}

// Serve runs the server
func (cs *CacheServer) Serve() {

	// let the middleware log.
	mux := http.NewServeMux()

	cs.SetLogger()
	if cs.logger != nil {
		mux.Handle(cachePath, serviceLoader(cacheIndexHandler(), requestMetrics(cs.logger)))
		mux.Handle(statsPath, serviceLoader(statsIndexHandler(), requestMetrics(cs.logger)))
		cs.logger.Printf("starting cache server on :%d", cs.Port)
	} else {
		mux.Handle(cachePath, serviceLoader(cacheIndexHandler()))
		mux.Handle(statsPath, serviceLoader(statsIndexHandler()))
	}

	Addr := ":" + strconv.Itoa(cs.Port)

	srv := &http.Server{
		Addr:    Addr,
		Handler: mux,
	}

	go srv.ListenAndServe()
	<-cs.Context.Ctx.Done() // wait for done call

	// shutdown
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("cache server Shutdown Failed:%+s", err)
	}
}

// Shutdown shuts down the server
func (cs *CacheServer) Shutdown() {
	cs.Context.Cancel()
}
