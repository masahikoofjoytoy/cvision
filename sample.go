package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"text/template"
)

type Category struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

type Categories []Category

type Description struct {
	Tags     []string `json:"tags"`
	Captions Captions `json:"captions"`
}

type Caption struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

type Captions []Caption

type Metadata struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

type Adult struct {
	IsAdultContent bool    `json:"isAdultContent"`
	IsRacyContent  bool    `json:"isRacyContent"`
	AdultScore     float64 `json:"adultScore"`
	RacyScore      float64 `json:"racyScore"`
}

type AnalyzeResult struct {
	Categories  Categories  `json:"categories"`
	Description Description `json:"description"`
	RequestId   string      `json:"requestId"`
	Metadata    Metadata    `json:"metadata"`
	Faces       []string    `json:"faces"`
	Adult       Adult       `json:"adult"`
}

const (
	VisionUrl          = "https://southeastasia.api.cognitive.microsoft.com/vision/v1.0/analyze?"
	VisionAccessKey    = "2cd669333bc04623a126161cdd2ade75"
	TranslateUrl       = "https://api.microsofttranslator.com/v2/http.svc/Translate?"
	TokenUrl           = "https://api.cognitive.microsoft.com/sts/v1.0/issueToken"
	TranslateAccessKey = "45201519841f4367817b8d0128165118"
	ResultFile         = "/home/hasegawa-ma/computerVision/sample/static/result.jpg"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", index)
	http.HandleFunc("/analyze", analyze)
	http.ListenAndServe(":8888", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	tpl, _ := template.ParseFiles("views/index.html")
	tpl.Execute(w, nil)
}

func analyze(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	defer file.Close()
	if err != nil {
		log.Println(err)
		return
	}

	updFile, err := os.Create(ResultFile)
	defer updFile.Close()
	if err != nil {
		log.Println(err)
		return
	}

	if _, err = io.Copy(updFile, file); err != nil {
		log.Println(err)
		return
	}

	readFile, err := os.Open(ResultFile)
	if err != nil {
		log.Println(err)
		return
	}

	data, err := ioutil.ReadAll(readFile)
	if err != nil {
		log.Println(err)
		return
	}

	analyzeResult, err := vision(bytes.NewReader(data))
	if err != nil {
		log.Println("error")
		return
	}
	translated, err := translate(analyzeResult.Description.Captions[0].Text)
	if err != nil {
		log.Println("error")
		return
	}

	tpl, _ := template.ParseFiles("views/analyze.html")
	tpl.Execute(w, map[string]string{"Translated": translated})
}

func vision(data *bytes.Reader) (analyzeResult *AnalyzeResult, err error) {
	values := url.Values{}
	values.Add("visualFeatures", "description,adult,categories")
	req, _ := http.NewRequest("POST", VisionUrl+values.Encode(), data)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Ocp-Apim-Subscription-Key", VisionAccessKey)
	client := new(http.Client)
	response, err := client.Do(req)
	if err != nil {
		log.Println("error")
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("error")
		return
	}
	analyzeResult = new(AnalyzeResult)
	err = json.Unmarshal(body, analyzeResult)
	return
}

func translate(translateText string) (translated string, err error) {
	req, err := http.NewRequest("POST", TokenUrl, nil)
	req.Header.Set("Ocp-Apim-Subscription-Key", TranslateAccessKey)
	if err != nil {
		return
	}

	client := new(http.Client)
	response, _ := client.Do(req)
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	appid := url.QueryEscape("Bearer " + string(body))
	text := url.QueryEscape(translateText)
	req, _ = http.NewRequest("GET", TranslateUrl+"from=en&to=ja&text="+text+"&appid="+appid, nil)

	response, _ = client.Do(req)
	defer response.Body.Close()
	body, _ = ioutil.ReadAll(response.Body)
	rep := regexp.MustCompile(`<("[^"]*"|'[^']*'|[^'">])*>`)
	translated = rep.ReplaceAllString(string(body), "")
	return
}
