package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
)

const (
	ListDir      = 0x001
	UPLOAD_DIR   = "./uploads"
	TEMPLATE_DIR = "./views"
)

var templates map[string]*template.Template = make(map[string]*template.Template)

func init() {
	files, err := ioutil.ReadDir(TEMPLATE_DIR)
	checkerr(err)

	var templateName, templatePath string
	for _, fileInfo := range files {
		templateName = fileInfo.Name()
		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}

		templatePath = TEMPLATE_DIR + "/" + templateName
		log.Println("Loading template:", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		templates[templateName] = t
	}
}

func checkerr(err error) {
	if err != nil {
		panic(err)
	}
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) {
	err := templates[tmpl+".html"].Execute(w, locals)
	checkerr(err)
}

func fileIsExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return os.IsExist(err)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request方法：", r.Method)

	if r.Method == "GET" {
		renderHtml(w, "upload", nil)
	}
	if r.Method == "POST" {
		file, filehead, err := r.FormFile("image")
		checkerr(err)

		filename := filehead.Filename
		fmt.Println("文件名称：", filename)
		defer file.Close()
		var uploadFile *os.File
		var err2 error
		uploadFile, err2 = ioutil.TempFile(UPLOAD_DIR, filename) //在上传文件存储目录创建一个文件
		checkerr(err2)

		filearr := strings.Split(uploadFile.Name(), "\\")
		savefilename := filearr[1]
		fmt.Println("文件路径：", savefilename)
		defer uploadFile.Close()

		_, err = io.Copy(uploadFile, file)
		checkerr(err)

		fmt.Println("开始重定向......")
		http.Redirect(w, r, "/view?id="+savefilename, http.StatusFound) //重定向到一个网址
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	fmt.Println("显示图片路径：", imagePath)
	if exists := fileIsExists(imagePath); !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(UPLOAD_DIR)
	checkerr(err)

	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range files {
		images = append(images, fileInfo.Name())
	}

	locals["images"] = images
	renderHtml(w, "list", locals)
}

func safeHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			var e error
			var ok bool
			if e, ok = recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				log.Println("WARN: panic in %v. - %v", handler, e)
				log.Println(string(debug.Stack()))
			}
		}()
		handler(w, r)
	}
}

func get(url string) {
	resp, err := http.Get(url)
	checkerr(err)

	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func main() {
	fmt.Println("Photo Websit")
	//get("http://news.163.com")

	mux := http.NewServeMux()
	mux.HandleFunc("/", safeHandler(listHandler))
	mux.HandleFunc("/view", safeHandler(viewHandler))
	mux.HandleFunc("/upload", safeHandler(uploadHandler))

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal("ListenAndServe:", err.Error())
	}
}
