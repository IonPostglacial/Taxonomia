package identification

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"nicolas.galipot.net/taxonomia/dataset"
	"nicolas.galipot.net/taxonomia/dataset/database"
)

type Handler struct {
	reg      *database.DatasetRegistry
	template *template.Template
	store    *sessions.CookieStore
}

type TemplateData struct {
	PickedCharacter  *dataset.Character
	AnsweredChars    []*dataset.Character
	AnsweredStates   []*dataset.State
	AnsweredCharIds  []string
	AnsweredStateIds []string
	IdentifiedTaxons []*dataset.Taxon
}

func NewHandler(reg *database.DatasetRegistry, sessionKey string) *Handler {
	tpl, err := template.ParseFiles("dataset/identification/identification.html")
	if err != nil {
		log.Fatalf("cannot parse template %q: %q", "identification", err.Error())
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
	if charId := r.Form["reset"]; len(charId) > 0 {
		answeredCharIds = nil
		answeredStateIds = nil
	} else if charId := r.Form["pass"]; len(charId) > 0 {
		// TODO: keep track of the fact we passed
	} else if charId := r.Form["cancel"]; len(charId) > 0 {
		if len(answeredCharIds) > 0 {
			answeredCharIds = answeredCharIds[:len(answeredCharIds)-1]
			answeredStateIds = answeredStateIds[:len(answeredStateIds)-1]
		}
	} else {
		if charId := r.Form["character-id"]; len(charId) > 0 {
			answeredCharIds = append(answeredCharIds, charId...)
		}
		if stateId := r.Form["state-id"]; len(stateId) > 0 {
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
	tplData := TemplateData{
		AnsweredChars:    answeredChars,
		AnsweredCharIds:  answeredCharIds,
		AnsweredStateIds: answeredStateIds,
		IdentifiedTaxons: taxons,
	}
	characters, err := h.reg.GetAllCharactersExcept(answeredCharIds)
	if err != nil {
		log.Fatalf("Cannot retrieve characters: %q.\n", err.Error())
	}
	for _, character := range characters {
		tplData.PickedCharacter = character
		break
	}
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.template.Execute(w, tplData)
}
