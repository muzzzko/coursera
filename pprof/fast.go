package main

import (
	"bufio"
	"fmt"
	"github.com/mailru/easyjson/jlexer"
	"io"
	"os"
	"strings"
)

type User struct {
	Email string `json:"email"`
	Job string `json:"job"`
	Name string `json:"name"`
	Country string `json:"country"`
	Company string `json:"company"`
	Phone string `json:"phone"`
	Browsers []string `json:"browsers"`
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	var line []byte

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "found users:")

	scanner := bufio.NewScanner(file)
	user := User{}
	isAndroid := false
	isMSIE := false
	found := true
	for count := 0;scanner.Scan();count++ {
		isAndroid = false
		isMSIE = false

		line = scanner.Bytes()

		user.UnmarshalEasyJSON(&jlexer.Lexer{Data: line})
		// fmt.Printf("%v %v\n", err, line)
		//err := json.Unmarshal(line, &user)
		//if err != nil {
		//	panic(err)
		//}

		for _, browser := range user.Browsers {
			found = false
			if strings.Contains(browser, "Android") {
				isAndroid = true
				found = true
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
				found = true
			}

			if found {
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
				found = false
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := strings.Replace(user.Email, "@", " [at] ", -1)
		fmt.Fprintf(out,"[%d] %s <%s>\n", count, user.Name, email)
	}
	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}