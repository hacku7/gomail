/**
* @author: [hk7]
* @Data:   17:57
 */

package utils

import (
	"bytes"
	"html/template"
	"net/http"

	l4g "github.com/alecthomas/log4go"
	"github.com/fsnotify/fsnotify"
	"github.com/nicksnyder/go-i18n/i18n"
)

var (
	DEFAULT_LOCALE = "zh-CN"

	htmlTemplates *template.Template
	T             i18n.TranslateFunc
	locales       map[string]string = make(map[string]string)
)

type HTMLTemplate struct {
	TemplateName string
	Props        map[string]interface{}
	Html         map[string]template.HTML
	Locale       string
}

func InitHTML() error {
	return InitHTMLWithDir("templates")
}

func InitHTMLWithDir(dir string) error {
	var err error
	templatesDir, _ := FindDir(dir)
	if htmlTemplates, err = template.ParseGlob(templatesDir + "*.html"); err != nil {
		l4g.Error(T("api.init.parsing_templates.error"), err)
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		l4g.Error(T("web.create_dir.error"), err)
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					l4g.Info(T("web.reparse_templates.info"), event.Name)
					if htmlTemplates, err = template.ParseGlob(templatesDir + "*.html"); err != nil {
						l4g.Error(T("web.parsing_templates.error"), err)
					}
				}
			case err := <-watcher.Errors:
				l4g.Error(T("web.dir_fail.error"), err)
			}
		}
	}()

	err = watcher.Add(templatesDir)
	if err != nil {
		l4g.Error(T("web.watcher_fail.error"), err)
		return err
	}

	return nil
}

func NewHTMLTemplate(templateName string, locale string) *HTMLTemplate {
	return &HTMLTemplate{
		TemplateName: templateName,
		Props:        make(map[string]interface{}),
		Html:         make(map[string]template.HTML),
		Locale:       locale,
	}
}

func GetUserTranslations(locale string) i18n.TranslateFunc {
	if _, ok := locales[locale]; !ok {
		locale = DEFAULT_LOCALE
	}

	translations := TFuncWithFallback(locale)
	return translations
}

func TFuncWithFallback(pref string) i18n.TranslateFunc {
	t, _ := i18n.Tfunc(pref)
	return func(translationID string, args ...interface{}) string {
		if translated := t(translationID, args...); translated != translationID {
			return translated
		}

		t, _ := i18n.Tfunc(DEFAULT_LOCALE)
		return t(translationID, args...)
	}
}

func (t *HTMLTemplate) addDefaultProps() {
	var localT i18n.TranslateFunc
	if len(t.Locale) > 0 {
		localT = GetUserTranslations(t.Locale)
	} else {
		localT = T
	}

	t.Props["Footer"] = localT("api.templates.email_footer")
	t.Props["Organization"] = ""
	t.Html["EmailInfo"] = template.HTML(localT("",
		map[string]interface{}{"SupportEmail": "", "SiteName": ""}))
}

func (t *HTMLTemplate) Render() string {
	t.addDefaultProps()

	var text bytes.Buffer

	if err := htmlTemplates.ExecuteTemplate(&text, t.TemplateName, t); err != nil {
		l4g.Error(T("api.render.error"), t.TemplateName, err)
	}

	return text.String()
}

func (t *HTMLTemplate) RenderToWriter(w http.ResponseWriter) error {
	t.addDefaultProps()

	if err := htmlTemplates.ExecuteTemplate(w, t.TemplateName, t); err != nil {
		l4g.Error(T("api.render.error"), t.TemplateName, err)
		return err
	}
	return nil
}
