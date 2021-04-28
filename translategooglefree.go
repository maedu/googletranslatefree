package translategooglefree

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/robertkrimen/otto"
)

type Sentence struct {
	Orig  string `json:"orig"`
	Trans string `json:"trans"`
}

type Dict struct {
	Pos      string   `json:"pos"`
	Terms    []string `json:"terms"`
	BaseForm string   `json:"base_form"`
}

type AlternativeTranslation struct {
	Alternative []Alternative `json:"alternative"`
}

type Alternative struct {
	WordPostproc string `json:"word_postproc"`
}

type Result struct {
	Sentences               []Sentence               `json:"sentences"`
	Dict                    []Dict                   `json:"dict"`
	AlternativeTranslations []AlternativeTranslation `json:"alternative_translations"`
}
type Translation struct {
	Orig         string
	Trans        string
	Alternatives []string
}

// javascript "encodeURI()"
// so we embed js to our golang programm
func encodeURI(s string) (string, error) {
	eUri := `eUri = encodeURI(sourceText);`
	vm := otto.New()
	err := vm.Set("sourceText", s)
	if err != nil {
		return "err", errors.New("Error setting js variable")
	}
	_, err = vm.Run(eUri)
	if err != nil {
		return "err", errors.New("Error executing jscript")
	}
	val, err := vm.Get("eUri")
	if err != nil {
		return "err", errors.New("Error getting variable value from js")
	}
	v, err := val.ToString()
	if err != nil {
		return "err", errors.New("Error converting js var to string")
	}
	return v, nil
}

func Translate(source, sourceLang, targetLang string) (Translation, error) {
	var result Result

	encodedSource, err := encodeURI(source)
	if err != nil {
		return Translation{}, err
	}
	url := "https://translate.googleapis.com/translate_a/single?client=gtx" +
		"&sl=" + sourceLang +
		"&tl=" + targetLang +
		"&q=" + encodedSource +
		"&dt=bd&dt=t&dt=at" +
		"&dj=1"

	// dt=t&dt=at&dj=1&dt=rm&dt=bd&dt=ss&dt=md&dt=ex&dt=rw

	r, err := http.Get(url)
	if err != nil {
		return Translation{}, fmt.Errorf("Error getting translate.googleapis.com: %w")
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Translation{}, fmt.Errorf("Error reading response body: %w")
	}

	bReq := strings.Contains(string(body), `<title>Error 400 (Bad Request)`)
	if bReq {
		return Translation{}, errors.New("Error 400 (Bad Request)")
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return Translation{}, fmt.Errorf("Error unmarshaling data: %w")
	}

	if len(result.Sentences) == 0 {
		return Translation{}, errors.New("No sentences")
	}

	origs := []string{}
	trans := []string{}

	for _, sentence := range result.Sentences {
		origs = append(origs, sentence.Orig)
		trans = append(trans, sentence.Trans)
	}

	translation := Translation{
		Orig:         strings.Join(origs, ""),
		Trans:        strings.Join(trans, ""),
		Alternatives: []string{},
	}

	for _, alternativeTranslation := range result.AlternativeTranslations {
		for _, alternative := range alternativeTranslation.Alternative {
			translation.Alternatives = append(translation.Alternatives, alternative.WordPostproc)
		}
	}

	return translation, nil

}
