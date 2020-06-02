# escpos
golang esc/pos Lib
## A quick Example
```golang
package main

import (
	"bufio"
	"bytes"
	"image"
	"os"
	"strings"

	escpos "escposTest/lib"
	"github.com/olekukonko/tablewriter"
	qrcode "github.com/skip2/go-qrcode"
)
// https://github.com/seer-robotics/escpos/blob/master/escpos.go
func main() {
	var png []byte
	f, err := os.OpenFile("/dev/ttyUSB0",os.O_RDWR,0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	p := escpos.New(w)

	p.Verbose = true

	p.Init()
	p.Beep(4)
	p.SetFontSize(2, 3)
	p.SetFont("B")
	p.SetReverse(0)
	p.WriteGBK("简体字转繁体字")
	p.SetFont("C")
	p.Write("test2")

	p.SetEmphasize(1)
	p.Write("hello")
	p.Formfeed()
	png, _ = qrcode.Encode("https://www.bing.com", qrcode.Low, 256)
	img, _, _ := image.Decode(bytes.NewReader(png))
	p.SetAlign("center")
	p.PrintImage(img)
	p.SetUnderline(1)
	p.SetFontSize(4, 4)
	p.Write("hello")
	p.SetReverse(1)
	p.SetFontSize(2, 4)
	p.Write("hello")
	p.FormfeedN(10)
	p.SetAlign("center")
	p.Write("test")
	p.Linefeed()



	p.SetEmphasize(0)
	p.SetReverse(0)
	p.SetFontSize(2, 2)
	p.SetUnderline(0)
	data := [][]string{
		[]string{"充值", "The Good", "500"},
		[]string{"找零", "The Ruby", "288"},
		[]string{"应收", "The Ugly", "120"},
		[]string{"实收", "The Gopher", "800"},
	}
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Sign", "Rating"})

	for _, v := range data {
		table.Append(v)
	}
	table.SetAutoFormatHeaders(true)
	table.SetAutoMergeCells(true)
	table.SetAutoWrapText(true)
	table.SetBorder(true)
	table.Render() // Send output
	p.WriteGBK(tableString.String())
	p.Linefeed()
	p.Write("test")
	p.FormfeedD(200)

	p.Cut()

	w.Flush()
}

```
## reference
+ [seer-robotics/escpos](https://github.com/seer-robotics/escpos/blob/master/escpos.go) （use its Chinese support,and main Code）
+ [panjjo/escpos](https://github.com/panjjo/escpos) (refer his print images)
