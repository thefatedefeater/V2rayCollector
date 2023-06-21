package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"bufio"
	"github.com/PuerkitoBio/goquery"
)

var client = &http.Client{}
func main() {
    configFile := "channels.txt"
    channels, err := readLines(configFile)
    if err != nil {
        log.Fatal(err)
    }

    configs := map[string]string{
        "ss":     "",
        "vmess":  "",
        "trojan": "",
        "vless":  "",
        "mixed":  "",
    }

    myregex := map[string]string{
        "ss":     `(.{3})ss:\/\/`,
        "vmess":  `vmess:\/\/`,
        "trojan": `trojan:\/\/`,
        "vless":  `vless:\/\/`,
    }

    for i := 0; i < len(channels); i++ {
        all_messages := false
        if strings.Contains(channels[i], "{all_messages}") {
            all_messages = true
            channels[i] = strings.Split(channels[i], "{all_messages}")[0]
        }
        fmt.Println(channels[i])

        doc, err := fetchDocument(channels[i])
        if err != nil {
            log.Fatal(err)
        }

        if shouldLoadMoreMessages(doc) {
            number := extractLastMessageNumber(doc)
            doc = getMoreMessages(100, doc, number, channels[i])
        }

        if all_messages {
            processAllMessages(doc, myregex, configs)
        } else {
            processCodeAndPreBlocks(doc, myregex, configs)
        }
    }

    for proto, configcontent := range configs {
        WriteToFile(RemoveDuplicate(configcontent), proto+"_iran.txt")
    }
}

func readLines(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return lines, scanner.Err()
}

func fetchDocument(url string) (*goquery.Document, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return nil, err
    }

    return doc, nil
}

func shouldLoadMoreMessages(doc *goquery.Document) bool {
    messages := doc.Find(".tgme_widget_message_wrap").Length()
    _, exist := doc.Find(".tgme_widget_message_wrap .js-widget_message").Last().Attr("data-post")
    return messages < 100 && exist == true
}

func extractLastMessageNumber(doc *goquery.Document) string {
    link, _ := doc.Find(".tgme_widget_message_wrap .js-widget_message").Last().Attr("data-post")
    return strings.Split(link, "/")[1]
}

func getMoreMessages(length int, doc *goquery.Document, number string, channel string) *goquery.Document {
    x := load_more(channel + "?before=" + number)

    html2, _ := x.Html()
    reader2 := strings.NewReader(html2)
    doc2, _ := goquery.NewDocumentFromReader(reader2)

    doc.Find("body").AppendSelection(doc2.Find("body").Children())

    newDoc := goquery.NewDocumentFromNode(doc.Selection.Nodes[0])
    messages := newDoc.Find(".js-widget_message_wrap").Length()

    if messages > length {
        return newDoc
    } else {
        num, _ := strconv.Atoi(number)
        n := num - 21
        if n > 0 {
            ns := strconv.Itoa(n)
            GetMessages(length, newDoc, ns, channel)
        } else {
            return newDoc
        }
    }

    return newDoc
}
func GetMessages(length int, doc *goquery.Document, number string, channel string) *goquery.Document {
	x := load_more(channel + "?before=" + number)

	html2, _ := x.Html()
	reader2 := strings.NewReader(html2)
	doc2, _ := goquery.NewDocumentFromReader(reader2)

	doc.Find("body").AppendSelection(doc2.Find("body").Children())

	newDoc := goquery.NewDocumentFromNode(doc.Selection.Nodes[0])
	messages := newDoc.Find(".js-widget_message_wrap").Length()

	if messages > length {
		return newDoc
	} else {
		num, _ := strconv.Atoi(number)
		n := num - 21
		if n > 0 {
			ns := strconv.Itoa(n)
			GetMessages(length, newDoc, ns, channel)
		} else {
			return newDoc
		}
	}

	return newDoc
}

func processAllMessages(doc *goquery.Document, myregex map[string]string, configs map[string]string) {
    doc.Find(".tgme_widget_message_text").Each(func(j int, s *goquery.Selection) {
        message_text := s.Text()
        lines := strings.Split(message_text, "\n")
        for a := 0; a < len(lines); a++ {
            for _, regex_value := range myregex {
                re := regexp.MustCompile(regex_value)
                lines[a] = re.ReplaceAllStringFunc(lines[a], func(match string) string {
                    return "\n" + match
                })
            }
            for proto, _ := range configs {
                if strings.Contains(lines[a], proto) {
                    configs["mixed"] += "\n" + lines[a] + "\n"
                }
            }
        }
    })
}

func processCodeAndPreBlocks(doc *goquery.Document, myregex map[string]string, configs map[string]string) {
    doc.Find("code,pre").Each(func(j int, s *goquery.Selection) {
        message_text := s.Text()
        lines := strings.Split(message_text, "\n")
        for a := 0; a < len(lines); a++ {
            for proto_regex, regex_value := range myregex {
                re := regexp.MustCompile(regex_value)
                lines[a] = re.ReplaceAllStringFunc(lines[a],func(match string) string {
                    return proto_regex + "://" + match
                })
            }
            for proto, _ := range configs {
                if strings.Contains(lines[a], proto+"://") {
                    configs[proto] += "\n" + lines[a] + "\n"
                }
            }
        }
    })
}

func load_more(url string) *goquery.Document {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        log.Fatal(err)
    }

    return doc
}

func RemoveDuplicate(str string) string {
    lines := strings.Split(str, "\n")
    unique := make(map[string]bool)
    var result []string
    for _, line := range lines {
        if _, ok := unique[line]; !ok {
            unique[line] = true
            result = append(result, line)
        }
    }
    return strings.Join(result, "\n")
}

func WriteToFile(content string, filename string) {
    file, err := os.Create(filename)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    _, err = file.WriteString(content)
    if err != nil {
        log.Fatal(err)
    }
}