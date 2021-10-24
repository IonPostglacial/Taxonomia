package identification

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "embed"

	"github.com/gorilla/sessions"
	"nicolas.galipot.net/taxonomia/dataset"
	"nicolas.galipot.net/taxonomia/dataset/database"
)

//go:embed header.html
var headerTemplateTxt string

//go:embed characters.html
var listCharactersTxt string

//go:embed identify.html
var identifyTemplateTxt string

type Handler struct {
	reg      *database.DatasetRegistry
	template *template.Template
	store    *sessions.CookieStore
}

type TemplateData struct {
	PickedCharacter  *dataset.Character
	UnansweredChars  []*dataset.Character
	AnsweredChars    []*dataset.Character
	AnsweredStates   []*dataset.State
	AnsweredCharIds  []string
	AnsweredStateIds []string
	IdentifiedTaxons []*dataset.Taxon
}

func NewHandler(reg *database.DatasetRegistry, sessionKey string) *Handler {
	tpl := template.New("identify")
	_, err := tpl.Parse(headerTemplateTxt)
	if err != nil {
		log.Fatalf("cannot parse template %q: %q", "header", err.Error())
	}
	_, err = tpl.Parse(listCharactersTxt)
	if err != nil {
		log.Fatalf("cannot parse template %q: %q", "characters", err.Error())
	}
	_, err = tpl.Parse(identifyTemplateTxt)
	if err != nil {
		log.Fatalf("cannot parse template %q: %q", "identify", err.Error())
	}
	return &Handler{reg: reg, template: tpl, store: sessions.NewCookieStore([]byte(sessionKey))}
}

func (h *Handler) Func(w http.ResponseWriter, r *http.Request) {
	session, _ := h.store.Get(r, "identification")
	answeredCharIds, _ := session.Values["characters"].([]string)
	answeredStateIds, _ := session.Values["states"].([]string)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(r.Form["action"]) > 0 {
		switch r.Form["action"][0] {
		case "reset":
			answeredCharIds = nil
			answeredStateIds = nil
		case "pass":
			if charId := r.Form["selected-character"]; len(charId) > 0 {
				answeredCharIds = append(answeredCharIds, charId...)
				answeredStateIds = append(answeredStateIds, "")
			}
		case "cancel":
			if len(answeredCharIds) > 0 {
				answeredCharIds = answeredCharIds[:len(answeredCharIds)-1]
				answeredStateIds = answeredStateIds[:len(answeredStateIds)-1]
			}
		}
	} else {
		if charId := r.Form["selected-character"]; len(charId) > 0 {
			answeredCharIds = append(answeredCharIds, charId...)
		}
		if stateId := r.Form["selected-state"]; len(stateId) > 0 {
			answeredStateIds = append(answeredStateIds, stateId...)
		}
	}
	session.Values["characters"] = answeredCharIds
	session.Values["states"] = answeredStateIds
	taxons := []*dataset.Taxon{}
	if len(answeredStateIds) > 0 {
		taxons, err = h.reg.GetTaxonsHavingStates(answeredStateIds)
	}
	answeredChars, err := h.reg.GetCharactersFromIds(answeredCharIds, answeredStateIds)
	if err != nil {
		log.Fatalf("Cannot retrieve characters: %q.\n", err.Error())
	}
	characters, charsByIds, err := h.reg.GetAllCharactersExcept(answeredCharIds)
	if err != nil {
		log.Fatalf("Cannot retrieve characters: %q.\n", err.Error())
	}
	tplData := TemplateData{
		UnansweredChars:  characters,
		AnsweredChars:    answeredChars,
		AnsweredCharIds:  answeredCharIds,
		AnsweredStateIds: answeredStateIds,
		IdentifiedTaxons: taxons,
	}
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query := r.URL.Query()
	if openCharacter := query.Get("in"); len(openCharacter) > 0 {
		fmt.Printf("%v\n", charsByIds[openCharacter].Children)
		if ch, ok := charsByIds[openCharacter]; ok {
			chars := make([]*dataset.Character, 0, len(ch.Children))
			for _, child := range ch.Children {
				if charChild, ok := charsByIds[child.Id]; ok {
					chars = append(chars, charChild)
				}
			}
			tplData.UnansweredChars = chars
		}
	}
	if pickedChar := query.Get("char"); len(pickedChar) > 0 {
		if pickedCharacted, ok := charsByIds[pickedChar]; ok {
			tplData.PickedCharacter = pickedCharacted
		} else if len(characters) > 0 {
			tplData.PickedCharacter = characters[0]
		}
		h.template.ExecuteTemplate(w, "identify", tplData)
	} else {
		h.template.ExecuteTemplate(w, "characters", tplData)
	}
}
