package server

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/coreos/dex/api"
	"github.com/coreos/pkg/capnslog"
	"github.com/coreos/pkg/health"
	"github.com/urfave/negroni"

	"github.com/openshift/console/pkg/auth"
	// "github.com/openshift/console/pkg/backend"
	helmhandlerspkg "github.com/openshift/console/pkg/helm/handlers"
	"github.com/openshift/console/pkg/proxy"
	"github.com/openshift/console/pkg/serverutils"
	"github.com/openshift/console/pkg/terminal"
	"github.com/openshift/console/pkg/version"

	hproxy "github.com/openshift/console/pkg/hypercloud/proxy"
)

const (
	indexPageTemplateName     = "index.html"
	tokenizerPageTemplateName = "tokener.html"

	authLoginEndpoint              = "/auth/login"
	AuthLoginCallbackEndpoint      = "/auth/callback"
	AuthLoginSuccessEndpoint       = "/"
	AuthLoginErrorEndpoint         = "/error"
	authLogoutEndpoint             = "/auth/logout"
	k8sProxyEndpoint               = "/api/kubernetes/"
	prometheusProxyEndpoint        = "/api/prometheus"
	prometheusTenancyProxyEndpoint = "/api/prometheus-tenancy"
	alertManagerProxyEndpoint      = "/api/alertmanager"
	meteringProxyEndpoint          = "/api/metering"
	customLogoEndpoint             = "/custom-logo"
	helmChartRepoProxyEndpoint     = "/api/helm/charts/"

	grafanaProxyEndpoint          = "/api/grafana/"
	kialiProxyEndpoint            = "/api/kiali/"
	webhookEndpoint               = "/api/webhook/"
	hypercloudServerEndpoint      = "/api/hypercloud/"
	multiHypercloudServerEndpoint = "/api/multi-hypercloud/"
	kibanaEndpoint                = "/api/kibana/"
	kubeflowEndpoint              = "/api/kubeflow/"
)

var (
	plog = capnslog.NewPackageLogger("github.com/openshift/console", "server")
)

type jsGlobals struct {
	ConsoleVersion string `json:"consoleVersion"`
	// AuthDisabled             bool   `json:"authDisabled"`
	// KubectlClientID          string `json:"kubectlClientID"`
	BasePath string `json:"basePath"`
	// LoginURL                 string `json:"loginURL"`
	// LoginSuccessURL          string `json:"loginSuccessURL"`
	// LoginErrorURL            string `json:"loginErrorURL"`
	// LogoutURL                string `json:"logoutURL"`
	// LogoutRedirect           string `json:"logoutRedirect"`
	// RequestTokenURL          string `json:"requestTokenURL"`
	// KubeAdminLogoutURL       string `json:"kubeAdminLogoutURL"`
	KubeAPIServerURL         string `json:"kubeAPIServerURL"`
	PrometheusBaseURL        string `json:"prometheusBaseURL"`
	PrometheusTenancyBaseURL string `json:"prometheusTenancyBaseURL"`
	AlertManagerBaseURL      string `json:"alertManagerBaseURL"`
	MeteringBaseURL          string `json:"meteringBaseURL"`
	Branding                 string `json:"branding"`
	CustomProductName        string `json:"customProductName"`
	// CustomLogoURL            string `json:"customLogoURL"`
	// StatuspageID             string `json:"statuspageID"`
	// DocumentationBaseURL     string `json:"documentationBaseURL"`
	// AlertManagerPublicURL    string `json:"alertManagerPublicURL"`
	GrafanaPublicURL    string `json:"grafanaPublicURL"`
	PrometheusPublicURL string `json:"prometheusPublicURL"`
	ThanosPublicURL     string `json:"thanosPublicURL"`
	LoadTestFactor      int    `json:"loadTestFactor"`
	GOARCH              string `json:"GOARCH"`
	GOOS                string `json:"GOOS"`

	KeycloakRealm           string `json:keycloakRealm`
	KeycloakAuthURL         string `json:keycloakAuthURL`
	KeycloakClientId        string `json:keycloakClientId`
	KeycloakUseHiddenIframe bool   `json:keycloakUseHiddenIframe`

	McMode         bool   `json:mcMode`
	McModeFile     string `json:mcModeFile`
	McModeOperator bool   `json:mcModeOperator`

	ReleaseModeFlag bool `json:"releaseModeFlag"`

	// gitalb 화면 url -> ui 에서 사용
	GitlabURL string `json:"gitlabURL"`
}

type Server struct {
	K8sProxyConfig       *proxy.Config
	BaseURL              *url.URL
	LogoutRedirect       *url.URL
	PublicDir            string
	TectonicVersion      string
	Auther               *auth.Authenticator
	StaticUser           *auth.User
	KubectlClientID      string
	KubeAPIServerURL     string
	DocumentationBaseURL *url.URL
	Branding             string
	CustomProductName    string
	CustomLogoFile       string
	StatuspageID         string
	LoadTestFactor       int
	DexClient            api.DexClient
	// A client with the correct TLS setup for communicating with the API server.
	K8sClient                *http.Client
	PrometheusProxyConfig    *proxy.Config
	ThanosProxyConfig        *proxy.Config
	ThanosTenancyProxyConfig *proxy.Config
	AlertManagerProxyConfig  *proxy.Config
	MeteringProxyConfig      *proxy.Config
	TerminalProxyTLSConfig   *tls.Config
	// A lister for resource listing of a particular kind
	MonitoringDashboardConfigMapLister ResourceLister
	KnativeEventSourceCRDLister        ResourceLister
	HelmChartRepoProxyConfig           *proxy.Config
	GOARCH                             string
	GOOS                               string
	// Monitoring and Logging related URLs
	AlertManagerPublicURL *url.URL
	GrafanaPublicURL      *url.URL
	PrometheusPublicURL   *url.URL
	ThanosPublicURL       *url.URL

	// Add keycloak info for auth
	KeycloakRealm           string
	KeycloakAuthURL         string
	KeycloakClientId        string
	KeycloakUseHiddenIframe bool

	// Add proxy config
	GrafanaProxyConfig               *hproxy.Config
	KialiProxyConfig                 *hproxy.Config
	WebhookProxyConfig               *hproxy.Config
	HypercloudServerProxyConfig      *hproxy.Config
	MultiHypercloudServerProxyConfig *hproxy.Config
	KibanaProxyConfig                *hproxy.Config
	KubeflowProxyConfig              *hproxy.Config

	McMode         bool
	McModeFile     string
	McModeOperator bool

	// console mode
	ReleaseModeFlag bool

	// gitlab url
	// gitalb 화면 url -> ui 에서 사용
	GitlabURL string
}

func (s *Server) authDisabled() bool {
	return s.Auther == nil
}

func (s *Server) prometheusProxyEnabled() bool {
	return s.PrometheusProxyConfig != nil && s.ThanosTenancyProxyConfig != nil
}

func (s *Server) alertManagerProxyEnabled() bool {
	return s.AlertManagerProxyConfig != nil
}

func (s *Server) meteringProxyEnabled() bool {
	return s.MeteringProxyConfig != nil
}

func (s *Server) grafanaEnable() bool {
	return s.GrafanaProxyConfig != nil
}

func (s *Server) kialiEnable() bool {
	return s.KialiProxyConfig != nil
}

func (s *Server) webhookEnable() bool {
	return s.WebhookProxyConfig != nil
}

func (s *Server) hypercloudServerEnable() bool {
	return s.HypercloudServerProxyConfig != nil
}

func (s *Server) multiHypercloudServerEnable() bool {
	return s.MultiHypercloudServerProxyConfig != nil
}

func (s *Server) kibanaEnable() bool {
	return s.KibanaProxyConfig != nil
}

func (s *Server) KubeflowEnable() bool {
	return s.KubeflowProxyConfig != nil
}

func (s *Server) HTTPHandler() http.Handler {
	mux := http.NewServeMux()

	if len(s.BaseURL.Scheme) > 0 && len(s.BaseURL.Host) > 0 {
		s.K8sProxyConfig.Origin = fmt.Sprintf("%s://%s", s.BaseURL.Scheme, s.BaseURL.Host)
	}
	handle := func(path string, handler http.Handler) {
		mux.Handle(proxy.SingleJoiningSlash(s.BaseURL.Path, path), handler)
	}

	handleFunc := func(path string, handler http.HandlerFunc) { handle(path, handler) }

	fn := func(loginInfo auth.LoginJSON, successURL string, w http.ResponseWriter) {
		jsg := struct {
			auth.LoginJSON    `json:",inline"`
			LoginSuccessURL   string `json:"loginSuccessURL"`
			Branding          string `json:"branding"`
			CustomProductName string `json:"customProductName"`
		}{
			LoginJSON:         loginInfo,
			LoginSuccessURL:   successURL,
			Branding:          s.Branding,
			CustomProductName: s.CustomProductName,
		}

		tpl := template.New(tokenizerPageTemplateName)
		tpl.Delims("[[", "]]")
		tpls, err := tpl.ParseFiles(path.Join(s.PublicDir, tokenizerPageTemplateName))
		if err != nil {
			fmt.Printf("%v not found in configured public-dir path: %v", tokenizerPageTemplateName, err)
			os.Exit(1)
		}

		if err := tpls.ExecuteTemplate(w, tokenizerPageTemplateName, jsg); err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
	}

	authHandler := func(hf http.HandlerFunc) http.Handler {
		return authMiddleware(s.Auther, hf)
	}
	authHandlerWithUser := func(hf func(*auth.User, http.ResponseWriter, *http.Request)) http.Handler {
		return authMiddlewareWithUser(s.Auther, hf)
	}

	if s.authDisabled() {
		authHandler = func(hf http.HandlerFunc) http.Handler {
			return hf
		}
		authHandlerWithUser = func(hf func(*auth.User, http.ResponseWriter, *http.Request)) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				// if s.StaticUser.Username == "hypercloud" {
				if s.ReleaseModeFlag {
					token := r.Header.Clone().Get("Authorization")
					temp := strings.Split(token, "Bearer ")
					if len(temp) > 1 {
						token = temp[1]
					} else {
						token = temp[0]
					}
					// plog.Infof("check token is on : %v", token)
					s.StaticUser.Token = token

					// NOTE: query에 token 정보가 있을 시 해당 token으로 설정
					queryToken := r.URL.Query().Get("token")
					if queryToken != "" && token == "" {
						r.URL.Query().Del("token")
						s.StaticUser.Token = queryToken
					}

				}

				hf(s.StaticUser, w, r)
			})
		}

	}

	if !s.authDisabled() {
		handleFunc(authLoginEndpoint, s.Auther.LoginFunc)
		handleFunc(authLogoutEndpoint, s.Auther.LogoutFunc)
		handleFunc(AuthLoginCallbackEndpoint, s.Auther.CallbackFunc(fn))

		handle("/api/openshift/delete-token", authHandlerWithUser(s.handleOpenShiftTokenDeletion))
	}

	handleFunc("/api/", notFoundHandler)

	staticHandler := http.StripPrefix(proxy.SingleJoiningSlash(s.BaseURL.Path, "/static/"), http.FileServer(http.Dir(s.PublicDir)))
	handle("/static/", gzipHandler(securityHeadersMiddleware(staticHandler)))

	k8sApiHandler := http.StripPrefix(proxy.SingleJoiningSlash(s.BaseURL.Path, "/api/resource/"), http.FileServer(http.Dir("./api")))
	handle("/api/resource/", gzipHandler(securityHeadersMiddleware(k8sApiHandler)))

	if s.CustomLogoFile != "" {
		handleFunc(customLogoEndpoint, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, s.CustomLogoFile)
		})
	}

	// Scope of Service Worker needs to be higher than the requests it is intercepting (https://stackoverflow.com/a/35780776/6909941)
	handleFunc("/load-test.sw.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(s.PublicDir, "load-test.sw.js"))
	})

	handleFunc("/health", health.Checker{
		Checks: []health.Checkable{},
	}.ServeHTTP)

	// TODO: K8S api
	k8sProxy := proxy.NewProxy(s.K8sProxyConfig)
	handle(k8sProxyEndpoint, http.StripPrefix(
		proxy.SingleJoiningSlash(s.BaseURL.Path, k8sProxyEndpoint),
		authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
			// plog.Infof("request Header token : %v", s.StaticUser.Token)
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.StaticUser.Token))
			k8sProxy.ServeHTTP(w, r)
		})),
	)

	terminalProxy := terminal.NewProxy(
		s.TerminalProxyTLSConfig,
		s.K8sProxyConfig.TLSClientConfig,
		s.K8sProxyConfig.Endpoint)

	handle(terminal.ProxyEndpoint, authHandlerWithUser(terminalProxy.HandleProxy))
	handleFunc(terminal.AvailableEndpoint, terminalProxy.HandleProxyEnabled)

	if s.prometheusProxyEnabled() {
		// Only proxy requests to the Prometheus API, not the UI.
		var (
			labelSourcePath      = prometheusProxyEndpoint + "/api/v1/label/"
			rulesSourcePath      = prometheusProxyEndpoint + "/api/v1/rules"
			querySourcePath      = prometheusProxyEndpoint + "/api/v1/query"
			queryRangeSourcePath = prometheusProxyEndpoint + "/api/v1/query_range"
			targetAPIPath        = prometheusProxyEndpoint + "/api/"

			tenancyQuerySourcePath      = prometheusTenancyProxyEndpoint + "/api/v1/query"
			tenancyQueryRangeSourcePath = prometheusTenancyProxyEndpoint + "/api/v1/query_range"
			tenancyTargetAPIPath        = prometheusTenancyProxyEndpoint + "/api/"

			prometheusProxy    = proxy.NewProxy(s.PrometheusProxyConfig)
			thanosProxy        = proxy.NewProxy(s.ThanosProxyConfig)
			thanosTenancyProxy = proxy.NewProxy(s.ThanosTenancyProxyConfig)
		)

		// global label, query, and query_range requests have to be proxied via thanos
		handle(querySourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, targetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				thanosProxy.ServeHTTP(w, r)
			})),
		)
		handle(queryRangeSourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, targetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				thanosProxy.ServeHTTP(w, r)
			})),
		)
		handle(labelSourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, targetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				thanosProxy.ServeHTTP(w, r)
			})),
		)

		// alerting (rules) have to be proxied via cluster monitoring prometheus
		handle(rulesSourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, targetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				prometheusProxy.ServeHTTP(w, r)
			})),
		)

		// tenancy queries and query ranges have to be proxied via thanos
		handle(tenancyQuerySourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, tenancyTargetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				thanosTenancyProxy.ServeHTTP(w, r)
			})),
		)
		handle(tenancyQueryRangeSourcePath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, tenancyTargetAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				thanosTenancyProxy.ServeHTTP(w, r)
			})),
		)
	}

	if s.alertManagerProxyEnabled() {
		alertManagerProxyAPIPath := alertManagerProxyEndpoint + "/api/"
		alertManagerProxy := proxy.NewProxy(s.AlertManagerProxyConfig)
		handle(alertManagerProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, alertManagerProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				alertManagerProxy.ServeHTTP(w, r)
			})),
		)
	}

	if s.meteringProxyEnabled() {
		meteringProxyAPIPath := meteringProxyEndpoint + "/api/"
		meteringProxy := proxy.NewProxy(s.MeteringProxyConfig)
		handle(meteringProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, meteringProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				meteringProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: grafa proxy
	if s.grafanaEnable() {
		grafanaProxyAPIPath := grafanaProxyEndpoint
		grafanaProxy := httputil.NewSingleHostReverseProxy(s.GrafanaProxyConfig.Endpoint)
		handle(grafanaProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, grafanaProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				// r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				// s.StaticUser.Token = r.Header.Clone().Get("Authorization")
				grafanaProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: kiali proxy
	if s.kialiEnable() {
		kialiProxyAPIPath := kialiProxyEndpoint
		// kialiProxy := httputil.NewSingleHostReverseProxy(s.KialiProxyConfig.Endpoint)
		kialiProxy := hproxy.NewProxy(s.KialiProxyConfig)
		handle(kialiProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, kialiProxyAPIPath),
			// 	http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 		kialiProxy.ServeHTTP(w, r)
			// 	})),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				// r.Header.Set("Host", s.KialiProxyConfig.Endpoint.Host)
				// r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				kialiProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: webhook proxy
	if s.webhookEnable() {
		webhookProxyAPIPath := webhookEndpoint
		webhookProxy := hproxy.NewProxy(s.WebhookProxyConfig)
		handle(webhookProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, webhookProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				webhookProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: hypercloudServer proxy
	if s.hypercloudServerEnable() {
		hypercloudServerProxyAPIPath := hypercloudServerEndpoint
		hypercloudServerProxy := hproxy.NewProxy(s.HypercloudServerProxyConfig)
		handle(hypercloudServerProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, hypercloudServerProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				hypercloudServerProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: multi-hypercloudServer proxy
	if s.multiHypercloudServerEnable() {
		multiHypercloudServerProxyAPIPath := multiHypercloudServerEndpoint
		multiHypercloudServerProxy := hproxy.NewProxy(s.MultiHypercloudServerProxyConfig)
		handle(multiHypercloudServerProxyAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, multiHypercloudServerProxyAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				multiHypercloudServerProxy.ServeHTTP(w, r)
			})),
		)
	}

	// NOTE: kibana proxy
	if s.kibanaEnable() {
		kibanaAPIPath := kibanaEndpoint
		kibanaProxy := hproxy.NewProxy(s.KibanaProxyConfig)
		// authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
		// 	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
		handle(kibanaAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, kibanaAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				kibanaProxy.ServeHTTP(w, r)
			})),
		)
	}

	if s.KubeflowEnable() {
		kubeflowAPIPath := kubeflowEndpoint
		kubeflowProxy := hproxy.NewProxy(s.KubeflowProxyConfig)
		handle(kubeflowAPIPath, http.StripPrefix(
			proxy.SingleJoiningSlash(s.BaseURL.Path, kubeflowAPIPath),
			authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
				r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
				kubeflowProxy.ServeHTTP(w, r)
			})),
		)
	}

	handle("/api/console/monitoring-dashboard-config", authHandler(s.handleMonitoringDashboardConfigmaps))
	handle("/api/console/knative-event-sources", authHandler(s.handleKnativeEventSourceCRDs))
	handle("/api/console/version", authHandler(s.versionHandler))

	// Helm Endpoints
	helmHandlers := helmhandlerspkg.New(s.KubeAPIServerURL, s.K8sClient.Transport)
	handle("/api/helm/template", authHandlerWithUser(helmHandlers.HandleHelmRenderManifests))
	handle("/api/helm/releases", authHandlerWithUser(helmHandlers.HandleHelmList))
	handle("/api/helm/chart", authHandlerWithUser(helmHandlers.HandleChartGet))
	handle("/api/helm/release/history", authHandlerWithUser(helmHandlers.HandleGetReleaseHistory))

	handle("/api/helm/release", authHandlerWithUser(func(user *auth.User, w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			helmHandlers.HandleGetRelease(user, w, r)
		case http.MethodPost:
			helmHandlers.HandleHelmInstall(user, w, r)
		case http.MethodDelete:
			helmHandlers.HandleUninstallRelease(user, w, r)
		case http.MethodPatch:
			helmHandlers.HandleRollbackRelease(user, w, r)
		case http.MethodPut:
			helmHandlers.HandleUpgradeRelease(user, w, r)
		default:
			w.Header().Set("Allow", "GET, POST, PATCH, PUT, DELETE")
			serverutils.SendResponse(w, http.StatusMethodNotAllowed, serverutils.ApiError{Err: "Unsupported method, supported methods are GET, POST, PATCH, PUT, DELETE"})
		}
	}))

	helmChartRepoProxy := proxy.NewProxy(s.HelmChartRepoProxyConfig)

	// Only proxy requests to chart repo index file
	handle(helmChartRepoProxyEndpoint+"index.yaml", http.StripPrefix(
		proxy.SingleJoiningSlash(s.BaseURL.Path, helmChartRepoProxyEndpoint),
		http.HandlerFunc(helmChartRepoProxy.ServeHTTP)))

	mux.HandleFunc(s.BaseURL.Path, s.indexHandler)
	n := negroni.New(negroni.NewLogger())
	n.UseHandler(mux)

	// return http.Handler(n)
	return securityHeadersMiddleware(http.Handler(n))
	// return securityHeadersMiddleware(http.Handler(mux))
}

func (s *Server) handleMonitoringDashboardConfigmaps(w http.ResponseWriter, r *http.Request) {
	s.MonitoringDashboardConfigMapLister.HandleResources(w, r)
}

func (s *Server) handleKnativeEventSourceCRDs(w http.ResponseWriter, r *http.Request) {
	s.KnativeEventSourceCRDLister.HandleResources(w, r)
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	jsg := &jsGlobals{
		ConsoleVersion: version.Version,
		// AuthDisabled:          s.authDisabled(),
		// KubectlClientID:       s.KubectlClientID,
		BasePath: s.BaseURL.Path,
		// LoginURL:              proxy.SingleJoiningSlash(s.BaseURL.String(), authLoginEndpoint),
		// LoginSuccessURL:       proxy.SingleJoiningSlash(s.BaseURL.String(), AuthLoginSuccessEndpoint),
		// LoginErrorURL:         proxy.SingleJoiningSlash(s.BaseURL.String(), AuthLoginErrorEndpoint),
		// LogoutURL:             proxy.SingleJoiningSlash(s.BaseURL.String(), authLogoutEndpoint),
		// LogoutRedirect:        s.LogoutRedirect.String(),
		KubeAPIServerURL:  s.KubeAPIServerURL,
		Branding:          s.Branding,
		CustomProductName: s.CustomProductName,
		// StatuspageID:          s.StatuspageID,
		// DocumentationBaseURL:  s.DocumentationBaseURL.String(),
		// AlertManagerPublicURL: s.AlertManagerPublicURL.String(),
		GrafanaPublicURL:    s.GrafanaPublicURL.String(),
		PrometheusPublicURL: s.PrometheusPublicURL.String(),
		ThanosPublicURL:     s.ThanosPublicURL.String(),

		GOARCH:         s.GOARCH,
		GOOS:           s.GOOS,
		LoadTestFactor: s.LoadTestFactor,

		// return ekycloak info
		KeycloakRealm:    s.KeycloakRealm,
		KeycloakAuthURL:  s.KeycloakAuthURL,
		KeycloakClientId: s.KeycloakClientId,

		McMode:         s.McMode,
		McModeFile:     s.McModeFile,
		McModeOperator: s.McModeOperator,

		ReleaseModeFlag: s.ReleaseModeFlag,
		GitlabURL:       s.GitlabURL,
	}

	// if !s.authDisabled() {
	// specialAuthURLs := s.Auther.GetSpecialURLs()
	// jsg.RequestTokenURL = specialAuthURLs.RequestToken
	// jsg.KubeAdminLogoutURL = specialAuthURLs.KubeAdminLogout
	// }

	if s.prometheusProxyEnabled() {
		jsg.PrometheusBaseURL = proxy.SingleJoiningSlash(s.BaseURL.Path, prometheusProxyEndpoint)
		jsg.PrometheusTenancyBaseURL = proxy.SingleJoiningSlash(s.BaseURL.Path, prometheusTenancyProxyEndpoint)
	}

	if s.alertManagerProxyEnabled() {
		jsg.AlertManagerBaseURL = proxy.SingleJoiningSlash(s.BaseURL.Path, alertManagerProxyEndpoint)
	}

	if s.meteringProxyEnabled() {
		jsg.MeteringBaseURL = proxy.SingleJoiningSlash(s.BaseURL.Path, meteringProxyEndpoint)
	}

	if !s.authDisabled() {
		s.Auther.SetCSRFCookie(s.BaseURL.Path, &w)
	}

	// if s.CustomLogoFile != "" {
	// jsg.CustomLogoURL = proxy.SingleJoiningSlash(s.BaseURL.Path, customLogoEndpoint)
	// }

	tpl := template.New(indexPageTemplateName)
	tpl.Delims("[[", "]]")
	tpls, err := tpl.ParseFiles(path.Join(s.PublicDir, indexPageTemplateName))
	if err != nil {
		fmt.Printf("index.html not found in configured public-dir path: %v", err)
		os.Exit(1)
	}

	if err := tpls.ExecuteTemplate(w, indexPageTemplateName, jsg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	serverutils.SendResponse(w, http.StatusOK, struct {
		Version string `json:"version"`
	}{
		Version: version.Version,
	})
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func (s *Server) handleOpenShiftTokenDeletion(user *auth.User, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		serverutils.SendResponse(w, http.StatusMethodNotAllowed, serverutils.ApiError{Err: "Invalid method: only POST is allowed"})
		return
	}

	// Delete the OpenShift OAuthAccessToken.
	path := "/apis/oauth.openshift.io/v1/oauthaccesstokens/" + user.Token
	url := proxy.SingleJoiningSlash(s.K8sProxyConfig.Endpoint.String(), path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		serverutils.SendResponse(w, http.StatusInternalServerError, serverutils.ApiError{Err: fmt.Sprintf("Failed to create token DELETE request: %v", err)})
		return
	}

	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
	resp, err := s.K8sClient.Do(req)
	if err != nil {
		serverutils.SendResponse(w, http.StatusBadGateway, serverutils.ApiError{Err: fmt.Sprintf("Failed to delete token: %v", err)})
		return
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()
}

// func (s *Server) ProxyHandler() http.Handler {
// 	plog.Info("is it working?")
// 	mux := gmux.NewRouter()
// 	mux.Path("/api/dynamic/test/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		plog.Info("is it a matching algorithm working?")
// 		w.Write([]byte("OK it is work"))
// 	})

// 	return http.Handler(mux)
// }
