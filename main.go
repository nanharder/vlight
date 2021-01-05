package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/parnurzeal/gorequest"
	"gopkg.in/gomail.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	FundJsUrl       = "http://fundgz.1234567.com.cn/js/"
	FundHTMLUrl     = "http://fund.eastmoney.com/"
	MIN_RISE_NUM    = 1.5
	MAX_FALL_NUM    = -1.5
)

var fundCodeSlice []string
var c conf

var dailyTitle = `
                 <tr>
	             <td width="50" align="center">基金名称</td>
	             <td width="50" align="center">估算涨幅</td>
	             <td width="50" align="center">当前估算净值</td>
	             <td width="50" align="center">昨日单位净值</td>
	             <td width="50" align="center">估算时间</td>
                 </tr>
                 `

var weeklyTitle = `
                 <tr>
	             <td width="50" align="center">基金名称</td>
	             <td width="50" align="center">近1周净值变化</td>
                 </tr>
                 `

var oneMonthTitle = `
                 <tr>
	             <td width="50" align="center">基金名称</td>
	             <td width="50" align="center">近1月净值变化</td>
                 </tr>
                 `

func FetchFund(codes []string) []map[string]string {
	var fundResult []map[string]string
	for _, code := range codes {
		fundDataMap := GetFundData(code)
		fundResult = append(fundResult, fundDataMap)
	}
	return fundResult
}

func GetFundData(code string) map[string]string {
	var weeklyChange string
	var oneMonthChange string
	fundJsUrl := FundJsUrl + code + ".js"
	request := gorequest.New()
	resp, body, err := request.Get(fundJsUrl).End()
	defer resp.Body.Close()
	if err != nil {
		log.Fatal(err)
		return nil
	}

	fundHTMLUrl := FundHTMLUrl + code + ".html"
	resp1, body1, err1 := request.Get(fundHTMLUrl).End()

	if err1 != nil {
		log.Fatal(err1)
		return nil
	}
	defer resp1.Body.Close()
	re, _ := regexp.Compile("jsonpgz\\((.*)\\);")
	ret := re.FindSubmatch([]byte(body))
	fundData := ret[1]

	doc, err2 := goquery.NewDocumentFromReader(strings.NewReader(body1))
	if err2 != nil {
		log.Fatal(err2)
	}
	doc.Find("#increaseAmount_stage > table:nth-child(1) > tbody:nth-child(1) > tr:nth-child(2)").Each(func(i int, s *goquery.Selection) {
		s.Find("td > div").Each(func(j int, k *goquery.Selection) {
			change := k.Text()
			switch j {
			case 1:
				weeklyChange = change
			case 2:
				oneMonthChange = change
			}
		})
	})
	var fundDataMap map[string]string
	if err := json.Unmarshal(fundData, &fundDataMap); err != nil {
		return nil
	}
	fundDataMap["weeklyChange"] = weeklyChange
	fundDataMap["oneMonthChange"] = oneMonthChange
	return fundDataMap
}

func GenerateHTML(fundResult []map[string]string) string {
	var dailyElements []string
	var dailyContent string
	var dailyChanges []string
	var dailyChangeContents string
	var weeklyElements []string
	var weeklyContent string
	var oneMonthElements []string
	var oneMonthContent string
	var dailyText string
	var weeklyText string
	var oneMonthText string
	now := time.Now()
	for _, fund := range fundResult {
		gszzl, err := strconv.ParseFloat(fund["gszzl"], 32)
		if err != nil {
			fmt.Printf("error: %s", err)
			continue
		}
		if gszzl > 0 {
			fund["gszzl"] = "+" + strconv.FormatFloat(gszzl, 'f', -1, 32)
		}
		// 每日涨幅通知
		dailyElement := `
                                   <tr>
                                     <td width="50" align="center">` + fund["name"] + `</td>
                                     <td width="50" align="center">` + fund["gszzl"] + `%</td>
                                     <td width="50" align="center">` + fund["gsz"] + `</td>
                                     <td width="50" align="center">` + fund["dwjz"] + `</td>
                                     <td width="50" align="center">` + fund["gztime"] + `</td>
                                   </tr>
	                           `
		dailyElements = append(dailyElements, dailyElement)

		dailyChange := `<img src="//j4.dfcfw.com/charts/pic6/` + fund["fundcode"] + `.png" alt="">`
		dailyChanges = append(dailyChanges, dailyChange)
		// 一周涨幅
		if now.Weekday().String() == c.WeekdayReport {
			weeklyElement := `
                                   <tr>
                                     <td width="50" align="center">` + fund["name"] + `</td>
                                     <td width="50" align="center">` + fund["weeklyChange"] + `</td>
                                   </tr>
                                   `
			weeklyElements = append(weeklyElements, weeklyElement)
		}
		// 月度涨幅
		if now.Day() == c.MonthdayReport {
			oneMonthElement := `
                                   <tr>
                                     <td width="50" align="center">` + fund["name"] + `</td>
                                     <td width="50" align="center">` + fund["oneMonthChange"] + `</td>
                                   </tr>
                                   `
			oneMonthElements = append(oneMonthElements, oneMonthElement)
		}
	}
	dailyContent = strings.Join(dailyElements, "\n")
	weeklyContent = strings.Join(weeklyElements, "\n")
	oneMonthContent = strings.Join(oneMonthElements, "\n")
	dailyChangeContents = strings.Join(dailyChanges, "\n")
	if dailyContent != "" {
		dailyText = `<table width="30%" border="1" cellspacing="0" cellpadding="0">` +
			dailyTitle + dailyContent + `</table> <br><br>`
	}
	if weeklyContent != "" {
		weeklyText = `<table width="30%" border="1" cellspacing="0" cellpadding="0">` +
			weeklyTitle + weeklyContent + `</table> <br><br>`
	}
	if oneMonthContent != "" {
		oneMonthText = `<table width="30%" border="1" cellspacing="0" cellpadding="0">` +
			oneMonthTitle + oneMonthContent + `</table> <br><br>`
	}
	html := `</html>
		       <head>
			    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		       </head>
                       <body>
			    <div id="container">
				<p>基金涨跌监控:</p>
			        <div id="content">
				            ` + dailyText + weeklyText + oneMonthText + `
				</div>
            	            </div>
			    <div id="container">
				<p>基金当日涨跌监控:</p>
			        <div id="content">
				            ` + dailyChangeContents + `
				</div>
            	            </div>
                       </body>
                 </html>`
	return html
}

func SendEmail(content string) {
	emailName := os.Getenv("EMAIL_NAME")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	m := gomail.NewMessage()
	m.SetHeader("From", emailName)
	m.SetHeader("To", emailName)
	m.SetHeader("Subject", "基金涨跌监控")
	m.SetBody("text/html", content)
	d := gomail.NewDialer("smtp.qq.com", 587, emailName, emailPassword)
	if err := d.DialAndSend(m); err != nil {
		log.Fatal(err)
	}
}

type conf struct {
	Code string `yaml:"code"`
	WeekdayReport string `yaml:"weekdayReport"`
	MonthdayReport int `yaml:"monthdayReport"`
	EmailName string `yaml:"emailName"`
	EmailPassword string `yaml:"emailPassword"`
}
func (c *conf) getConf() *conf {
	yamlFile, err := ioutil.ReadFile("settings.yml")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

func main() {
	c.getConf()
	fundCodeSlice = strings.Split(c.Code, ",")
	fundResult := FetchFund(fundCodeSlice)
	content := GenerateHTML(fundResult)
	SendEmail(content)
}
