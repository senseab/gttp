package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"git.wetofu.top/tonychee7000/gocolorcon"
)

const (
	defaultBindIP        = "[::]"
	defaultBindPort      = 8080
	defaultWorkDirectory = "/html"
	defaultIndex         = "index.html"
	defaultHTTPPath      = "/"
	gttpPrefix           = "GTTP_"
	envBindIP            = gttpPrefix + "BIND_IP"
	envBindPort          = gttpPrefix + "BIND_PORT"
	envWorkDirectory     = gttpPrefix + "WORK_DIR"
	envHTTPPath          = gttpPrefix + "HTTPPath"
	envSPAEnabled        = gttpPrefix + "SPA_ENABLED" // 1 or 0
	envIndex             = gttpPrefix + "INDEX"
	envSPAStaticPaths    = gttpPrefix + "SPA_STATIC_PATHS" // comma seprated.
)

var (
	bindIP            string
	bindPort          int
	workDirectory     string
	httpPath          string
	spaEnabled        bool
	index             string
	spaStaticPath     string
	spaStaticPathList []string
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LstdFlags | log.Lshortfile)
	log.SetPrefix(gocolorcon.SetColor(gocolorcon.ModeBold, gocolorcon.Green, gocolorcon.Default) + "GTTP.main" + gocolorcon.Clear() + "\t")
	flag.StringVar(&bindIP, "listen", defaultBindIP, "Bind IP, if IPv6, use [].")
	flag.IntVar(&bindPort, "port", defaultBindPort, "Listen at which port.")
	flag.StringVar(&httpPath, "http-path", defaultHTTPPath, "HTTP root path.")
	flag.StringVar(&workDirectory, "workdir", defaultWorkDirectory, "Specifiy dir contains html files.")
	flag.BoolVar(&spaEnabled, "use-spa", false, "Use single page appliction.")
	flag.StringVar(&index, "index", defaultIndex, "Set index file.")
	flag.StringVar(&spaStaticPath, "spa-static-path", "", "Which directories should be loaded globally. comma seprated.")
	flag.Parse()
	if !strings.HasPrefix(httpPath, "/") {
		log.Fatalln("Argument http-path should start with slash! current", httpPath)
	}
	if spaEnabled && spaStaticPath != "" {
		spaStaticPathList = strings.Split(spaStaticPath, ",")
		for _, url := range spaStaticPathList {
			if !strings.HasPrefix(url, "/") {
				log.Fatalln("Static paths should start with slash! current", url)
			}
		}
	}
}

func main() {
	var err error
	log.Println("Hello!")
	if _bindIP := os.Getenv(envBindIP); _bindIP != "" && bindIP == defaultBindIP {
		bindIP = _bindIP
	}
	if _port := os.Getenv(envBindPort); _port != "" && bindPort == defaultBindPort {
		bindPort, err = strconv.Atoi(_port)
		if err != nil {
			log.Println("Ignore bad environment variable", envBindPort, "with value:", _port+", use default port.")
			bindPort = defaultBindPort
		}
	}
	serveAt := fmt.Sprintf("%s:%d", bindIP, bindPort)
	log.Println("Will listen at:", serveAt)
	http.HandleFunc(httpPath, httpHandler)
	err = http.ListenAndServe(serveAt, nil)
	if err != nil {
		log.Fatalln("Fatal error! ", err)
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	const regexpFormat = "^.+(%s.*)$"
	var (
		status    = http.StatusOK
		urlPath   = r.URL.Path
		_buf      []byte
		buffer    = bytes.NewBuffer(_buf)
		forwarded = false
	)
	if r.Method != http.MethodGet {
		status = http.StatusMethodNotAllowed
		w.WriteHeader(status)
		buffer.WriteTo(w)
		return
	}
	if urlPath == httpPath {
		urlPath = index
	}

	if spaEnabled {
		for _, staticPath := range spaStaticPathList {
			if reg, err := regexp.Compile(fmt.Sprintf(regexpFormat, staticPath)); err != nil {
				log.Println("Construct regexp failed,", err)
				continue
			} else if reg.MatchString(urlPath) {
				urlPath = reg.ReplaceAllString(urlPath, "$1")
			}
		}
	}
	f, err := os.Open(path.Join(workDirectory, urlPath))
	if err != nil {
		switch {
		case os.IsNotExist(err):
			if spaEnabled {
				http.ServeFile(w, r, path.Join(workDirectory, index))
				forwarded = true
			} else {
				status = http.StatusNotFound
			}
		case os.IsPermission(err):
			status = http.StatusForbidden
		default:
			status = http.StatusInternalServerError
			log.Println("Error:", err)
		}
	}
	if f != nil {
		defer f.Close()
		buffer.ReadFrom(f)
	}
	log.Println("Access", r.RemoteAddr, r.Method, urlPath, status, "\""+r.UserAgent()+"\"", r.Referer(), r.Header.Get("X-Forwarded-For"))
	if !forwarded {
		defer buffer.WriteTo(w)
		defer w.WriteHeader(status)
	}
}
