package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

//PreviewImage represents a preview image for a page
type PreviewImage struct {
	URL       string `json:"url,omitempty"`
	SecureURL string `json:"secureURL,omitempty"`
	Type      string `json:"type,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Alt       string `json:"alt,omitempty"`
}

//PageSummary represents summary properties for a web page
type PageSummary struct {
	Type        string          `json:"type,omitempty"`
	URL         string          `json:"url,omitempty"`
	Title       string          `json:"title,omitempty"`
	SiteName    string          `json:"siteName,omitempty"`
	Description string          `json:"description,omitempty"`
	Author      string          `json:"author,omitempty"`
	Keywords    []string        `json:"keywords,omitempty"`
	Icon        *PreviewImage   `json:"icon,omitempty"`
	Images      []*PreviewImage `json:"images,omitempty"`
}

//SummaryHandler handles requests for the page summary API.
//This API expects one query string parameter named `url`,
//which should contain a URL to a web page. It responds with
//a JSON-encoded PageSummary struct containing the page summary
//meta-data.
func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	url := r.URL.Query()["url"]
	// if not supplied, respond with an http.StatusBadRequest error
	if url != nil {

		// call fetchHTML and extractSummary functions
		body, err := fetchHTML(strings.Join(url, ""))
		if err == nil {

			// get the summary struct
			summary, err2 := extractSummary(strings.Join(url, ""), body)
			if err2 != nil {
				enc := json.NewEncoder(os.Stdout)
				if err3 := enc.Encode(r); err3 == nil {
					fmt.Printf("error encoding struct into JSON: %v\n", err3)

				} else {

					w.WriteHeader(http.StatusOK)
					myJson, err := json.Marshal(summary)
					fmt.Println(myJson)
					if err != nil {
						panic(err)
					} else {

						w.Write(myJson)
					}

				}
			}
		} else {
			// respond with error
			http.Error(w, "Failed to get data from the provided URL.", 200)
		}

	} else {
		// respond with error
		http.Error(w, "Please supply a URL.", 401)
	}
}

//fetchHTML fetches `pageURL` and returns the body stream or an error.
//Errors are returned if the response status code is an error (>=400),
//or if the content type indicates the URL is not an HTML page.
func fetchHTML(pageURL string) (io.ReadCloser, error) {

	// GET the URL
	resp, err := http.Get(pageURL)

	// if there was an error, report it and exit
	if err != nil {
		return nil, fmt.Errorf("error fetching URL: %v\n", err)
	}

	// check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status code was %d\n", resp.StatusCode)
	}

	// check response content type
	ctype := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("response content type was %s not text/html\n", ctype)
	}

	return resp.Body, nil
}

func extractSummary(pageURL string, htmlStream io.ReadCloser) (*PageSummary, error) {

	// create a new tokenizer over the response body
	tokenizer := html.NewTokenizer(htmlStream)

	// summary struct
	summary := PageSummary{}

	// images struct
	images := []*PreviewImage{}

	// make sure the response body gets closed
	defer htmlStream.Close()

	// loop until we find the title element or encounter and error
	for {
		// get the next token type
		tokenType := tokenizer.Next()

		// if it's an error, we either
		// reached the end
		// html was malformed
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				// end of file, break out of the loop
				break
			}
			// html was malformed
			return nil, fmt.Errorf("error tokenizing HTML: %v", tokenizer.Err())
		}

		// if this is a start tag token or a self closing token
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {

			// get the token
			token := tokenizer.Token()

			// if the name of the element is "meta"
			if "meta" == token.Data {
				for i := 0; i < len(token.Attr); i++ {
					a := token.Attr[i]
					if a.Key == "property" {
						if a.Val == "og:type" {
							summary.Type = getContentAttrVal(token)
						} else if a.Val == "og:url" {
							summary.URL = getContentAttrVal(token)
						} else if a.Val == "og:title" {
							summary.Title = getContentAttrVal(token)
						} else if a.Val == "og:site_name" {
							summary.SiteName = getContentAttrVal(token)
						} else if a.Val == "og:description" {
							summary.Description = getContentAttrVal(token)
						} else if a.Val == "og:image" {
							// found preview image

							pimage := PreviewImage{}

							// if the url for the image doesn't contain http
							// then it's a relative url that needs to be absolute
							if !strings.Contains(getContentAttrVal(token), "http") {

								// strip url to base url
								index := strings.Index(pageURL, "com")
								strippedURL := pageURL[:index+3]

								pimage.URL = strippedURL + getContentAttrVal(token)
							} else {
								pimage.URL = getContentAttrVal(token)
							}
							images = append(images, &pimage)
						} else if a.Val == "og:image:width" {
							number, _ := strconv.Atoi(getContentAttrVal(token))
							images[len(images)-1].Width = number
						} else if a.Val == "og:image:height" {
							number, _ := strconv.Atoi(getContentAttrVal(token))
							images[len(images)-1].Height = number
						} else if a.Val == "og:image:type" {
							images[len(images)-1].Type = getContentAttrVal(token)
						} else if a.Val == "og:image:secure_url" {
							images[len(images)-1].SecureURL = getContentAttrVal(token)
						} else if a.Val == "og:image:alt" {
							images[len(images)-1].Alt = getContentAttrVal(token)
						}
					} else if a.Key == "name" && a.Val == "description" && summary.Description == "" {
						summary.Description = getContentAttrVal(token)
					} else if a.Key == "name" && a.Val == "author" {
						summary.Author = getContentAttrVal(token)
					} else if a.Key == "name" && a.Val == "keywords" {
						s := strings.Split(getContentAttrVal(token), ",")
						for _, str := range s {
							fmt.Println(str)
							summary.Keywords = append(summary.Keywords, strings.TrimSpace(str))
						}
					}
				}
				// If we come across a title tag and we haven't set the title yet...
			} else if "title" == token.Data && summary.Title == "" {
				//the next token should be the page title
				tokenType = tokenizer.Next()
				//just make sure it's actually a text token
				if tokenType == html.TextToken {
					//report the page title and break out of the loop
					summary.Title = tokenizer.Token().Data
				}
			} else if "link" == token.Data {
				linkIsIcon := false
				icon := PreviewImage{}
				for i := 0; i < len(token.Attr); i++ {
					a := token.Attr[i]

					if a.Key == "rel" && a.Val == "icon" {
						linkIsIcon = true
					} else if a.Key == "href" && linkIsIcon {

						// if the url for the image doesn't contain http
						// then it's a relative url that needs to be absolute
						if !strings.Contains(a.Val, "http") {

							// strip url to base url
							index := strings.Index(pageURL, "com")
							strippedURL := pageURL[:index+3]

							icon.URL = strippedURL + a.Val
						} else {
							icon.URL = a.Val
						}

					} else if a.Key == "sizes" && linkIsIcon {
						if a.Val != "any" {
							s := strings.Split(a.Val, "x")
							height, _ := strconv.Atoi(s[0])
							width, _ := strconv.Atoi(s[1])
							icon.Width = width
							icon.Height = height
						}
					} else if a.Key == "type" {
						icon.Type = a.Val
					}
				}
				summary.Icon = &icon
			}
		}
	}
	if len(images) > 0 {
		summary.Images = images
	} else {
		summary.Images = nil
	}
	return &summary, nil
}

func getContentAttrVal(token html.Token) string {
	for j := 0; j < len(token.Attr); j++ {
		if token.Attr[j].Key == "content" {
			return token.Attr[j].Val
		}
	}
	return ""
}
