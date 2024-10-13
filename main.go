// Copyright 2024 ryan.
// SPDX-License-Identifier: MIT

package main

import (
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const DVCON_DOCS_DIR = "dvcon-proceedings"

func saveDocument(url, filename string) error {
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch URL %s: %v", url, err)
		return err
	}
	defer res.Body.Close()

	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filename, err)
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, res.Body)
	if err != nil {
		log.Printf("Failed to copy content to file %s: %v", filename, err)
	}
	return err
}

type Document struct {
	URL      string
	Filename string
}

func getAllDocuments(doc *html.Node) []Document {
	var docs []Document
	// 递归遍历DOM的函数
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var doc Document
			for _, a := range n.Attr {
				if a.Key == "href" {
					doc.URL = a.Val
				}
				if a.Key == "download" {
					doc.Filename = a.Val
				}
				if doc.URL != "" && doc.Filename != "" {
					docs = append(docs, doc)
					break
				}
			}
			// if doc.URL == "" || doc.Filename == "" {
			// 	log.Printf("Failed to find URL or filename in <a> tag: %v", n.Attr)
			// }
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// 遍历DOM
	traverse(doc)
	return docs
}

// isDirExists 检查目录是否存在
func isDirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func getFilesInDir(dir string) []string {
	if !isDirExists(dir) {
		return nil
	}
	files, err := filepath.Glob(dir + "/*")
	if err != nil {
		log.Printf("Failed to get files in directory %s: %v", dir, err)
		return nil
	}
	return files
}

func main() {
	if !isDirExists(DVCON_DOCS_DIR) {
		if err := os.Mkdir(DVCON_DOCS_DIR, 0755); err != nil {
			log.Printf("Failed to create directory: %v", err)
			return
		}
	}

	// 获取HTML文档
	res, err := http.Get("https://dvcon-proceedings.org/document-search/")
	if err != nil {
		log.Printf("Failed to fetch URL: %v", err)
		return
	}
	defer res.Body.Close()

	// 解析HTML文档
	dvDoc, err := html.Parse(res.Body)
	if err != nil {
		log.Printf("Failed to parse HTML: %v", err)
		return
	}

	oldDocs := getFilesInDir(DVCON_DOCS_DIR)
	oldDocsMap := make(map[string]struct{})
	for _, f := range oldDocs {
		oldDocsMap[filepath.Base(f)] = struct{}{}
	}

	log.Printf("Found %d documents in the old directory", len(oldDocs))
	log.Printf("Found %d documents in the new directory", len(oldDocsMap))
	log.Println("Start downloading...")

	docs := getAllDocuments(dvDoc)
	var newDownloads int
	for _, doc := range docs {
		if _, ok := oldDocsMap[doc.Filename]; ok {
			log.Printf("Skip downloading %s, already exists in the old directory", doc.Filename)
			continue
		}
		err := saveDocument(doc.URL, filepath.Join(DVCON_DOCS_DIR, doc.Filename))
		if err != nil {
			log.Printf("Failed to download %s: %v", doc.Filename, err)
		} else {
			oldDocsMap[doc.Filename] = struct{}{}
			newDownloads++
			log.Printf("Downloaded %s", doc.Filename)
		}
	}
	log.Printf("Downloaded %d documents", newDownloads)
}
