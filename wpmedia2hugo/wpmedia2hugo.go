package main

import (
	"fmt"

	"regexp"

	"github.com/beevik/etree"
	"github.com/mvdan/xurls"
)

type Query struct {
	ItemList []Item `xml:"item"`
}

type Item struct {
	Title string `xml:"title"`
	//Redir Redirect `xml:"redirect"`
	Link string `xml:"link"`
}

func main() {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile("wordpress.xml"); err != nil {
		panic(err)
	}

	// Couldn't find how to go deeper in one command
	first := doc.SelectElement("rss")
	root := first.SelectElement("channel")

	for _, post := range root.SelectElements("item") {
		fmt.Println("+++")
		if title := post.SelectElement("title"); title != nil {
			fmt.Printf("title = \"%s\" \n", title.Text())
		}
		fmt.Println("draft = false")
		if date := post.SelectElement("wp:post_date"); date != nil {
			fmt.Printf("date = \"%s\" \n", date.Text())
		}
		var allTags string
		tagSayisi := post.SelectElements("category")
		for _, cat := range tagSayisi {
			if category := post.SelectElement("category"); category != nil {
				isTag := cat.SelectAttrValue("domain", "unknown")
				if isTag == "post_tag" {
					tags := cat.SelectAttrValue("nicename", "unknown")
					//fmt.Printf("  TAGS: %s \n", tags)
					if allTags == "" {
						allTags += tags
					} else {
						allTags += ", " + tags
					}
				}
			}
		}
		fmt.Printf("Tags = [%s] \n", allTags)
		fmt.Println()
		fmt.Println("+++")
		if content := post.SelectElement("content:encoded"); content != nil {
			//re := regexp.MustCompile("(http|ftp|https)://.*\\.(gif|jpg|jpeg|webm)")
			findContent := xurls.Relaxed.FindAllString(content.Text(), -1)
			if len(findContent) == 0 {
				re := regexp.MustCompile(".*(youtube|liveleak).*")
				if len(findContent) != 0 {
					findContent[0] = re.FindString(content.Text())
				}
			}
			if len(findContent) != 0 {
				wantedContent := findContent[len(findContent)-1]
				isImage, _ := regexp.MatchString(".*(gif|jpg|jpeg|png)\\?.*$", wantedContent)
				if isImage {
					re := regexp.MustCompile(".*(gif|jpg|jpeg|png)")
					realImg := re.FindString(wantedContent)
					fmt.Printf("<img src=\"%s\"></img> \n", realImg)
				}
				isVideo, _ := regexp.MatchString(".*(mp4|webm)$", wantedContent)
				if isVideo {
					fmt.Printf("<video controls autoplay loop src=\"%s\"></video> \n", wantedContent)
				}
				fmt.Println("  *** METACONTENT: " + findContent[len(findContent)-1])
			}
		}
		for _, attr := range post.Attr {
			fmt.Printf("  ATTR: %s=%s\n", attr.Key, attr.Value)
		}
		fmt.Println()
	}
}

