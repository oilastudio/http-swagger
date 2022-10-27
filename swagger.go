package httpSwagger

import (
	"golang.org/x/net/webdav"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"

	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/swag"
)

// WrapHandler wraps swaggerFiles.Handler and returns http.HandlerFunc.
var WrapHandler = Handler()

// Config stores httpSwagger configuration variables.
type Config struct {
	// The url pointing to API definition (normally swagger.json or swagger.yaml). Default is `doc.json`.
	URL                  string
	DocExpansion         string
	DomID                string
	InstanceName         string
	BeforeScript         template.JS
	AfterScript          template.JS
	Plugins              []template.JS
	UIConfig             map[template.JS]template.JS
	DeepLinking          bool
	PersistAuthorization bool
	Handler              *webdav.Handler
}

func WebdavHandler(handler *webdav.Handler) func(*Config) {
	return func(c *Config) {
		c.Handler = handler
	}
}

// URL presents the url pointing to API definition (normally swagger.json or swagger.yaml).
func URL(url string) func(*Config) {
	return func(c *Config) {
		c.URL = url
	}
}

// DeepLinking true, false.
func DeepLinking(deepLinking bool) func(*Config) {
	return func(c *Config) {
		c.DeepLinking = deepLinking
	}
}

// DocExpansion list, full, none.
func DocExpansion(docExpansion string) func(*Config) {
	return func(c *Config) {
		c.DocExpansion = docExpansion
	}
}

// DomID #swagger-ui.
func DomID(domID string) func(*Config) {
	return func(c *Config) {
		c.DomID = domID
	}
}

// InstanceName set the instance name that was used to generate the swagger documents
// Defaults to swag.Name ("swagger").
func InstanceName(name string) func(*Config) {
	return func(c *Config) {
		c.InstanceName = name
	}
}

// PersistAuthorization Persist authorization information over browser close/refresh.
// Defaults to false.
func PersistAuthorization(persistAuthorization bool) func(*Config) {
	return func(c *Config) {
		c.PersistAuthorization = persistAuthorization
	}
}

// Plugins specifies additional plugins to load into Swagger UI.
func Plugins(plugins []string) func(*Config) {
	return func(c *Config) {
		vs := make([]template.JS, len(plugins))
		for i, v := range plugins {
			vs[i] = template.JS(v)
		}
		c.Plugins = vs
	}
}

// UIConfig specifies additional SwaggerUIBundle config object properties.
func UIConfig(props map[string]string) func(*Config) {
	return func(c *Config) {
		vs := make(map[template.JS]template.JS, len(props))
		for k, v := range props {
			vs[template.JS(k)] = template.JS(v)
		}
		c.UIConfig = vs
	}
}

// BeforeScript holds JavaScript to be run right before the Swagger UI object is created.
func BeforeScript(js string) func(*Config) {
	return func(c *Config) {
		c.BeforeScript = template.JS(js)
	}
}

// AfterScript holds JavaScript to be run right after the Swagger UI object is created
// and set on the window.
func AfterScript(js string) func(*Config) {
	return func(c *Config) {
		c.AfterScript = template.JS(js)
	}
}

func newConfig(configFns ...func(*Config)) *Config {
	config := Config{
		URL:                  "doc.json",
		DocExpansion:         "list",
		DomID:                "swagger-ui",
		InstanceName:         "swagger",
		DeepLinking:          true,
		PersistAuthorization: false,
	}

	for _, fn := range configFns {
		fn(&config)
	}

	if config.InstanceName == "" {
		config.InstanceName = swag.Name
	}

	return &config
}

// Handler wraps `http.Handler` into `http.HandlerFunc`.
func Handler(configFns ...func(*Config)) http.HandlerFunc {
	var once sync.Once

	config := newConfig(configFns...)

	// create a template with name
	index, _ := template.New("swagger_index.html").Parse(indexTempl)

	re := regexp.MustCompile(`^(.*/)([^?].*)?[?|.]*$`)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		matches := re.FindStringSubmatch(r.RequestURI)

		path := matches[2]

		if config.Handler == nil {
			config.Handler = swaggerFiles.Handler
		}

		once.Do(func() {
			config.Handler.Prefix = matches[1]
		})

		switch filepath.Ext(path) {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}

		switch path {
		case "index.html":
			_ = index.Execute(w, config)
		case "doc.json":
			doc, err := swag.ReadDoc(config.InstanceName)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

				return
			}

			_, _ = w.Write([]byte(doc))
		case "":
			http.Redirect(w, r, config.Handler.Prefix+"index.html", http.StatusMovedPermanently)
		default:
			config.Handler.ServeHTTP(w, r)
		}
	}
}

const indexTempl = `<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html>
<head>
    <title>Swagger explorer</title>

    <meta name="viewport" content="width=device-width, initial-scale=1.0">

	<link rel="stylesheet" type="text/css" href="./swagger-ui.css" />
    <link rel="stylesheet" type="text/css" href="index.css" />

    <link href="css/index.css" rel="stylesheet"/>
    <link href='css/standalone.css' rel='stylesheet'/>
    <link href='css/api-explorer.css' rel='stylesheet' type='text/css'/>
    <link href='css/screen.css' media='screen' rel='stylesheet' type='text/css'/>

    <script src='lib/jquery-1.8.0.min.js' type='text/javascript'></script>
    <script src='lib/jquery.slideto.min.js' type='text/javascript'></script>
    <script src='lib/jquery.wiggle.min.js' type='text/javascript'></script>
    <script src='lib/jquery.ba-bbq.min.js' type='text/javascript'></script>
    <script src='lib/handlebars-2.0.0.js' type='text/javascript'></script>
    <script src='lib/underscore-min.js' type='text/javascript'></script>
    <script src='lib/backbone-min.js' type='text/javascript'></script>
    <script src='swagger-ui.js' type='text/javascript'></script>
    <script src='lib/jsoneditor.js' type='text/javascript'></script>
    <script src='lib/highlight.7.3.pack.js' type='text/javascript'></script>
    <script src='lib/marked.js' type='text/javascript'></script>
    <script src='lib/swagger-oauth.js' type='text/javascript'></script>
    <script src='lib/bootstrap.min.js' type='text/javascript'></script>

    <script type="text/javascript">
        jQuery.browser = jQuery.browser || {};
        (function () {
            jQuery.browser.msie = jQuery.browser.msie || false;
            jQuery.browser.version = jQuery.browser.version || 0;
            if (navigator.userAgent.match(/MSIE ([0-9]+)\./)) {
                jQuery.browser.msie = true;
                jQuery.browser.version = RegExp.$1;
            }
        })();
    </script>

    <script type="text/javascript">
        $(function () {
            var url = window.location.search.match(/url=([^&]+)/);
            if (url && url.length > 1) {
                url = decodeURIComponent(url[1]);
            } else {
                url = window.location.toString().replace(/\/*#?.*/, '/swagger.json');
            }

			  {{- if .BeforeScript}}
			  {{.BeforeScript}}
			  {{- end}}
			  // Build a system
			  const ui = SwaggerUIBundle({
				url: "{{.URL}}",
				deepLinking: {{.DeepLinking}},
				docExpansion: "{{.DocExpansion}}",
				dom_id: "#{{.DomID}}",
				persistAuthorization: {{.PersistAuthorization}},
				validatorUrl: null,
				presets: [
				  SwaggerUIBundle.presets.apis,
				  SwaggerUIStandalonePreset
				],
				plugins: [
				  SwaggerUIBundle.plugins.DownloadUrl
				  {{- range $plugin := .Plugins }},
				  {{$plugin}}
				  {{- end}}
				],
				{{- range $k, $v := .UIConfig}}
				{{$k}}: {{$v}},
				{{- end}}
				layout: "StandaloneLayout"
			  })
			
			  window.ui = ui
			  {{- if .AfterScript}}
			  {{.AfterScript}}
			  {{- end}}
        });
    </script>

    <script type="text/javascript">

        $(function () {

            $(window).scroll(function () {
                var sticky = $(".sticky-nav");

                i(sticky);
                r(sticky);

                function n() {
                    return window.matchMedia("(min-width: 992px)").matches
                }

                function e() {
                    n() ? sticky.parents(".sticky-nav-placeholder").removeAttr("style") : sticky.parents(".sticky-nav-placeholder").css("min-height", sticky.outerHeight())
                }

                function i(n) {
                    n.hasClass("fixed") || (navOffset = n.offset().top);
                    e();
                    $(window).scrollTop() > navOffset ? $(".modal.in").length || n.addClass("fixed") : n.removeClass("fixed")
                }

                function r(e) {
                    function i() {
                        var i = $(window).scrollTop(), r = e.parents(".sticky-nav");
                        return r.hasClass("fixed") && !n() && (i = i + r.outerHeight() + 40), i
                    }

                    function r(e) {
                        var t = o.next("[data-endpoint]"), n = o.prev("[data-endpoint]");
                        return "next" === e ? t.length ? t : o.parent().next().find("[data-endpoint]").first() : "prev" === e ? n.length ? n : o.parent().prev().find("[data-endpoint]").last() : []
                    }

                    var nav = e.find("[data-navigator]");
                    if (nav.find("[data-endpoint][data-selected]").length) {
                        var o = nav.find("[data-endpoint][data-selected]"),
                            a = $("#" + o.attr("data-endpoint")),
                            s = a.offset().top,
                            l = (s + a.outerHeight(), r("next")),
                            u = r("prev");
                        if (l.length) {
                            {
                                var d = $("#" + l.attr("data-endpoint")), f = d.offset().top;
                                f + d.outerHeight()
                            }
                            i() >= f && c(l)
                        }
                        if (u.length) {
                            var p = $("#" + u.attr("data-endpoint")),
                            g = u.offset().top;
                            v = (g + p.outerHeight(), 100);
                            i() < s - v && c(u)
                        }
                    }
                }

                function s() {
                    var e = $(".sticky-nav [data-navigator]"),
                        n = e.find("[data-endpoint]").first();
                    n.attr("data-selected", "");
                    u.find("[data-selected-value]").html(n.text())
                }

                function c(e) {
                    {
                        var n = $(".sticky-nav [data-navigator]");
                        $("#" + e.attr("data-endpoint"))
                    }
                    n.find("[data-resource]").removeClass("active");
                    n.find("[data-selected]").removeAttr("data-selected");
                    e.closest("[data-resource]").addClass("active");
                    e.attr("data-selected", "");
                    sticky.find("[data-selected-value]").html(e.text())
                }
            });

        });
    </script>

    <script type="text/javascript">
        $(function () {
            $("[data-toggle='tooltip']").tooltip();
        });
    </script>

</head>

<body class="page-docs" style="zoom: 1;">
<header class="site-header">
    <nav role="navigation" class="navbar navbar-default">
        <div class="container">
            <div class="navbar-header">
                <button type="button" data-toggle="collapse" data-target="#navbar-collapse" class="navbar-toggle"><span
                        class="sr-only">Toggle navigation</span><span class="icon-bar"></span><span
                        class="icon-bar"></span><span class="icon-bar"></span></button>
                <h1 class="navbar-brand"><a href="http://swagger.io"><span>swagger explorer</span></a></h1>
            </div>
            <div id="navbar-collapse" class="collapse navbar-collapse">
                <ul class="nav navbar-nav navbar-left">
                    <li class="li-why">
                      <a href="http://swagger.io"><strong>Swagger</strong> explorer</a>
                    </li>

                </ul>
            </div>
        </div>
    </nav>
</header>

<section class="content">
    <div id="api2-explorer">
        <div class="swagger-section page-docs" style="zoom: 1">
            <div class="main-section">
                <div id="swagger-ui-container" class="swagger-ui-wrap">
                </div>
            </div>
        </div>
    </div>
</section>
<script src="./swagger-ui-bundle.js" charset="UTF-8"> </script>
<script src="./swagger-ui-standalone-preset.js" charset="UTF-8"> </script>
<script src="./swagger-initializer.js" charset="UTF-8"> </script>
</body>
</html>
`

const indexTempl_bk = `<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="./swagger-ui.css" >
  <link rel="icon" type="image/png" href="./favicon-32x32.png" sizes="32x32" />
  <link rel="icon" type="image/png" href="./favicon-16x16.png" sizes="16x16" />
  <style>
    html
    {
        box-sizing: border-box;
        overflow: -moz-scrollbars-vertical;
        overflow-y: scroll;
    }
    *,
    *:before,
    *:after
    {
        box-sizing: inherit;
    }

    body {
      margin:0;
      background: #fafafa;
    }
  </style>
</head>

<body>

<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" style="position:absolute;width:0;height:0">
  <defs>
    <symbol viewBox="0 0 20 20" id="unlocked">
      <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V6h2v-.801C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8z"></path>
    </symbol>

    <symbol viewBox="0 0 20 20" id="locked">
      <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8zM12 8H8V5.199C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8z"/>
    </symbol>

    <symbol viewBox="0 0 20 20" id="close">
      <path d="M14.348 14.849c-.469.469-1.229.469-1.697 0L10 11.819l-2.651 3.029c-.469.469-1.229.469-1.697 0-.469-.469-.469-1.229 0-1.697l2.758-3.15-2.759-3.152c-.469-.469-.469-1.228 0-1.697.469-.469 1.228-.469 1.697 0L10 8.183l2.651-3.031c.469-.469 1.228-.469 1.697 0 .469.469.469 1.229 0 1.697l-2.758 3.152 2.758 3.15c.469.469.469 1.229 0 1.698z"/>
    </symbol>

    <symbol viewBox="0 0 20 20" id="large-arrow">
      <path d="M13.25 10L6.109 2.58c-.268-.27-.268-.707 0-.979.268-.27.701-.27.969 0l7.83 7.908c.268.271.268.709 0 .979l-7.83 7.908c-.268.271-.701.27-.969 0-.268-.269-.268-.707 0-.979L13.25 10z"/>
    </symbol>

    <symbol viewBox="0 0 20 20" id="large-arrow-down">
      <path d="M17.418 6.109c.272-.268.709-.268.979 0s.271.701 0 .969l-7.908 7.83c-.27.268-.707.268-.979 0l-7.908-7.83c-.27-.268-.27-.701 0-.969.271-.268.709-.268.979 0L10 13.25l7.418-7.141z"/>
    </symbol>

    <symbol viewBox="0 0 24 24" id="jump-to">
      <path d="M19 7v4H5.83l3.58-3.59L8 6l-6 6 6 6 1.41-1.41L5.83 13H21V7z"/>
    </symbol>

    <symbol viewBox="0 0 24 24" id="expand">
      <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
    </symbol>
  </defs>
</svg>

<div id="swagger-ui"></div>

<script src="./swagger-ui-bundle.js"> </script>
<script src="./swagger-ui-standalone-preset.js"> </script>
<script>
window.onload = function() {
  {{- if .BeforeScript}}
  {{.BeforeScript}}
  {{- end}}
  // Build a system
  const ui = SwaggerUIBundle({
    url: "{{.URL}}",
    deepLinking: {{.DeepLinking}},
    docExpansion: "{{.DocExpansion}}",
    dom_id: "#{{.DomID}}",
    persistAuthorization: {{.PersistAuthorization}},
    validatorUrl: null,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
      {{- range $plugin := .Plugins }},
      {{$plugin}}
      {{- end}}
    ],
    {{- range $k, $v := .UIConfig}}
    {{$k}}: {{$v}},
    {{- end}}
    layout: "StandaloneLayout"
  })

  window.ui = ui
  {{- if .AfterScript}}
  {{.AfterScript}}
  {{- end}}
}
</script>
</body>

</html>
`
