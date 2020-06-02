package escpos

import (
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/qiniu/iconv"
)


// text replacement map
var textReplaceMap = map[string]string{
	// horizontal tab
	"&#9;":  "\x09",
	"&#x9;": "\x09",

	// linefeed
	"&#10;": "\n",
	"&#xA;": "\n",

	// xml stuff
	"&apos;": "'",
	"&quot;": `"`,
	"&gt;":   ">",
	"&lt;":   "<",

	// ampersand must be last to avoid double decoding
	"&amp;": "&",
}

// replace text from the above map
func textReplace(data string) string {
	for k, v := range textReplaceMap {
		data = strings.Replace(data, k, v, -1)
	}
	return data
}

// Escpos struct
type Escpos struct {
	// destination
	dst io.Writer

	// font metrics
	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	// state toggles GS[char]
	reverse uint8

	Verbose bool
}

// reset toggles
func (e *Escpos) reset() {
	e.width = 1
	e.height = 1

	e.underline = 0
	e.emphasize = 0
	e.upsidedown = 0
	e.rotate = 0

	e.reverse = 0
}

// New create a Escpos printer
func New(dst io.Writer) (e *Escpos) {
	e = &Escpos{dst: dst}
	e.reset()
	return
}

// WriteRaw write raw bytes to printer
func (e *Escpos) WriteRaw(data []byte) (n int, err error) {
	if len(data) > 0 {
		if e.Verbose {
			log.Println("Writing %d bytes: %s\n", len(data), data)
		}
		e.dst.Write(data)
	} else {
		if e.Verbose {
			log.Println("Wrote NO bytes\n")
		}
	}

	return 0, nil
}

// Write a string to the printer
func (e *Escpos) Write(data string) (int, error) {
	return e.WriteRaw([]byte(data))
}

// WriteGBK write a string to the printer with GBK encode
func (e *Escpos) WriteGBK(data string) (int, error) {
	cd, err := iconv.Open("gbk", "utf-8")
	if err != nil {
		log.Println("iconv.Open failed!")
		return 0, err
	}
	defer cd.Close()
	gbk := cd.ConvString(data)
	return e.WriteRaw([]byte(gbk))
}

// WriteWEU write a string to the printer with Western European encode
func (e *Escpos) WriteWEU(data string) (int, error) {
	cd, err := iconv.Open("cp850", "utf-8")
	if err != nil {
		log.Println("iconv.Open failed!")
		return 0, err
	}
	defer cd.Close()
	weu := cd.ConvString(data)
	return e.WriteRaw([]byte(weu))
}

// Init printer settings
// \x1B@ => ESC @  初始化打印机
func (e *Escpos) Init() {
	e.reset()
	e.Write("\x1B@")
}

// Cut the paper
// \x1DVA0 => GS V A 0
func (e *Escpos) Cut() {
	e.Write("\x1DVA0")
}

// BanFeedButton 禁止面板按键
// \x1Bc5n => ESC c 5 n  n= 0, 1(禁止)
func (e *Escpos) BanFeedButton(n uint8) {
	s := string([]byte{'\x1B', 'c', '5', n})
	e.Write(s)
}

// Beep ...
// \x1BBnt => ESC B n t 蜂鸣器 n 为次数
func (e *Escpos) Beep(n uint8) {
	s := string([]byte{'\x1B', 'B', n, 9})
	e.Write(s)
}

// Linefeed ...
// 换行
func (e *Escpos) Linefeed() {
	e.Write("\n")
}

// FormfeedD ...
// \x1BJn => ESC J n 打印并进纸n*0.125mm 0<=n<=255
func (e *Escpos) FormfeedD(n uint8) {
	if n < 0 {
		n = 0
	} else if n > 255 {
		n = 255
	}
	s := string([]byte{'\x1B', 'J', n})
	e.Write(s)
}

// FormfeedN ...
// \x1Bdn => ESC d n 打印并进纸n行 0<=n<=255
func (e *Escpos) FormfeedN(n uint8) {
	if n < 0 {
		n = 0
	} else if n > 255 {
		n = 255
	}
	s := string([]byte{'\x1B', 'J', n})
	e.Write(s)
}

// Formfeed ...
// 打印并进纸1行
func (e *Escpos) Formfeed() {
	e.FormfeedN(1)
}

// SetFont ...
// \x1BMn => ESC M n  选择字型 A(12*24) B(9*17) C(don't know)
func (e *Escpos) SetFont(font string) {
	f := 0

	switch font {
	case "A":
		f = 0
	case "B":
		f = 1
	case "C":
		f = 2
	default:
		log.Println(fmt.Sprintf("Invalid font: '%s', defaulting to 'A'", font))
		f = 0
	}

	e.Write(fmt.Sprintf("\x1BM%c", f))
}

func (e *Escpos) sendFontSize() {
	s := string([]byte{'\x1D', '!', ((e.width - 1) << 4) | (e.height - 1)})
	e.Write(s)
}

// SetFontSize ...
// \x1D!n => GS ! n  设定字符大小
// 高度大于5倍时，打印机会挂掉，不知道为什么
func (e *Escpos) SetFontSize(width, height uint8) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		if height > 5 {
			height = 5
			log.Println("change height to 5, because height larger than 5 may cause some error")
		}
		e.width = width
		e.height = height
		e.sendFontSize()
	} else {
		log.Println(fmt.Sprintf("Invalid font size passed: %d x %d", width, height))
	}
}

func (e *Escpos) sendUnderline() {
	s := string([]byte{'\x1B', '-', e.underline})
	e.Write(s)
}

func (e *Escpos) sendEmphasize() {
	s := string([]byte{'\x1B', 'E', e.emphasize})
	e.Write(s)
}

func (e *Escpos) sendUpsidedown() {
	s := string([]byte{'\x1B', '{', e.upsidedown})
	e.Write(s)
}

func (e *Escpos) sendRotate() {
	s := string([]byte{'\x1B', 'V', e.rotate})
	e.Write(s)
}

func (e *Escpos) sendReverse() {
	s := string([]byte{'\x1D', 'B', e.reverse})
	e.Write(s)
}

func (e *Escpos) sendMoveX(x uint16) {
	e.Write(string([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)}))
}

func (e *Escpos) sendMoveY(y uint16) {
	e.Write(string([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)}))
}

// SetUnderline ...
// \x1B-n => ESC - n  设定/解除下划线 n = 0(解除), 1(1点粗), 2(2点粗)
func (e *Escpos) SetUnderline(v uint8) {
	e.underline = v
	e.sendUnderline()
}

// SetEmphasize ...
// \x1BGn => ESC E n  设定/解除粗体打印 n = 0, 1
func (e *Escpos) SetEmphasize(u uint8) {
	e.emphasize = u
	e.sendEmphasize()
}

// SetUpsidedown ...
// \x1B{n => ESC { n  设置/解除颠倒打印模式 n = 0, 1
func (e *Escpos) SetUpsidedown(v uint8) {
	e.upsidedown = v
	e.sendUpsidedown()
}

// SetRotate ...
// \x1BVn => ESC V n  字符180度旋转
func (e *Escpos) SetRotate(v uint8) {
	e.rotate = v
	e.sendRotate()
}

// SetReverse ...
// GS B n  设定/解除反白打印模式  n = 0, 1
func (e *Escpos) SetReverse(v uint8) {
	e.reverse = v
	e.sendReverse()
}

// SetMoveX ...
// \x1B$nLnH => ESC $ nL nH  x方向绝对定位
func (e *Escpos) SetMoveX(x uint16) {
	e.sendMoveX(x)
}

// Pulse (open the drawer)
// 发送脉冲，用来打开钱箱
func (e *Escpos) Pulse() {
	// with t=2 -- meaning 2*2msec
	e.Write("\x1Bp\x02")
}

// SetLineSpace ...
// \x1B3n => ESC 3 n  行间距n*0.125mm
func (e *Escpos) SetLineSpace(n ...uint8) {
	var s string
	switch len(n) {
	case 0:
		s = string([]byte{'\x1B', '2'})
	case 1:
		s = string([]byte{'\x1B', '3', n[0]})
	default:
		log.Println("Invalid num of params, using first param")
		s = string([]byte{'\x1B', '3', n[0]})
	}
	e.Write(s)
}

// SetAlign ...
// \x1Ban => ESC a n  选择对齐方式
func (e *Escpos) SetAlign(align string) {
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	default:
		log.Println(fmt.Sprintf("Invalid alignment: %s", align))
	}
	e.Write(fmt.Sprintf("\x1Ba%c", a))
}

// Text ...
func (e *Escpos) Text(params map[string]string, data string) {

	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// set emphasize
	if em, ok := params["em"]; ok && (em == "true" || em == "1") {
		e.SetEmphasize(1)
	}

	// set underline
	if ul, ok := params["ul"]; ok && (ul == "true" || ul == "1") {
		e.SetUnderline(1)
	}

	// set reverse
	if reverse, ok := params["reverse"]; ok && (reverse == "true" || reverse == "1") {
		e.SetReverse(1)
	}

	// set rotate
	if rotate, ok := params["rotate"]; ok && (rotate == "true" || rotate == "1") {
		e.SetRotate(1)
	}

	// set font
	if font, ok := params["font"]; ok {
		e.SetFont(strings.ToUpper(font[5:6]))
	}

	// do dw (double font width)
	if dw, ok := params["dw"]; ok && (dw == "true" || dw == "1") {
		e.SetFontSize(2, e.height)
	}

	// do dh (double font height)
	if dh, ok := params["dh"]; ok && (dh == "true" || dh == "1") {
		e.SetFontSize(e.width, 2)
	}

	// do font width
	if width, ok := params["width"]; ok {
		if i, err := strconv.Atoi(width); err == nil {
			e.SetFontSize(uint8(i), e.height)
		} else {
			log.Println(fmt.Sprintf("Invalid font width: %s", width))
		}
	}

	// do font height
	if height, ok := params["height"]; ok {
		if i, err := strconv.Atoi(height); err == nil {
			e.SetFontSize(e.width, uint8(i))
		} else {
			log.Println(fmt.Sprintf("Invalid font height: %s", height))
		}
	}

	// do y positioning
	if x, ok := params["x"]; ok {
		if i, err := strconv.Atoi(x); err == nil {
			e.sendMoveX(uint16(i))
		} else {
			log.Println("Invalid x param %d", x)
		}
	}

	// do y positioning
	if y, ok := params["y"]; ok {
		if i, err := strconv.Atoi(y); err == nil {
			e.sendMoveY(uint16(i))
		} else {
			log.Println("Invalid y param %d", y)
		}
	}

	// do text replace, then write data
	data = textReplace(data)
	if len(data) > 0 {
		e.Write(data)
	}
}

// Feed ...
func (e *Escpos) Feed(params map[string]string) {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			e.FormfeedN(uint8(i))
		} else {
			log.Println(fmt.Sprintf("Invalid line number %s", l))
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			e.sendMoveY(uint16(i))
		} else {
			log.Println(fmt.Sprintf("Invalid unit number %s", u))
		}
	}

	// send linefeed
	e.Linefeed()

	// reset variables
	e.reset()

	// reset printer
	e.sendEmphasize()
	e.sendRotate()
	e.sendReverse()
	e.sendUnderline()
	e.sendUpsidedown()
	e.sendFontSize()
}

// FeedAndCut ...
func (e *Escpos) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		e.Formfeed()
	}

	e.Cut()
}

// used to send graphics headers
func (e *Escpos) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	e.Write("\x1b(L")
	e.WriteRaw([]byte{byte(l % 256), byte(l / 256), m, fn})
	e.WriteRaw(data)
}

// Image write an image
func (e *Escpos) Image(params map[string]string, data string) {
	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// get width
	wstr, ok := params["width"]
	if !ok {
		log.Println("No width specified on image")
	}

	// get height
	hstr, ok := params["height"]
	if !ok {
		log.Println("No height specified on image")
	}

	// convert width
	width, err := strconv.Atoi(wstr)
	if err != nil {
		log.Println("Invalid image width %s", wstr)
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		log.Println("Invalid image height %s", hstr)
	}

	// decode data frome b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Println(err.Error())
	}

	if e.Verbose {
		log.Println("Image len:%d w: %d h: %d\n", len(dec), width, height)
	}

	header := []byte{
		byte('0'), 0x01, 0x01, byte('1'),
	}

	a := append(header, dec...)

	e.gSend(byte('0'), byte('p'), a)
	e.gSend(byte('0'), byte('2'), []byte{})

}

// WriteNode write a "node" to the printer
func (e *Escpos) WriteNode(name string, params map[string]string, data string) {
	cstr := ""
	if data != "" {
		str := data[:]
		if len(data) > 40 {
			str = fmt.Sprintf("%s ...", data[0:40])
		}
		cstr = fmt.Sprintf(" => '%s'", str)
	}

	if e.Verbose {
		log.Println("Write: %s => %+v%s\n", name, params, cstr)
	}

	switch name {
	case "text":
		e.Text(params, data)
	case "feed":
		e.Feed(params)
	case "cut":
		e.FeedAndCut(params)
	case "pulse":
		e.Pulse()
	case "image":
		e.Image(params, data)
	}
}


func (e *Escpos) SetPrintPic() {
	e.Write(fmt.Sprintf("\x1D*%c%c%v", 2, 2, "11111000001010101111100000101010"))
}

//打印下载位图
func (e *Escpos) PrintPic() {
	e.Write(fmt.Sprintf("\x1D/%c", 0))
}

func (e *Escpos) Barcode(s string) {
	e.Write(fmt.Sprintf("\x1Dk%c%s%c", 4, s, 0))
	// e.Write(fmt.Sprintf("\x1Dk%c%c%s", 4, len(s), s))
	// e.Write(fmt.Sprintf("\x1Dk%c12345678910%c", 1, 0))
}

//print barcode HRI
func (e *Escpos) BarcodeHRI(n int) {
	//0:no 1:top 2:down 3:top&&down
	e.Write(fmt.Sprintf("\x1DH%c", n))
}

//print barcode HRI font size
func (e *Escpos) BarcodeHRIFontSize(n int) {
	//0:A(12*24),1:B(9*17)
	e.Write(fmt.Sprintf("\x1Df%c", n))
}

//print barcode HRI font H
func (e *Escpos) BarcodeHigth(n int) {
	e.Write(fmt.Sprintf("\x1Dh%c", n))
}

func (e *Escpos) PrintImage(img image.Image) {
	e.SetLineSpace(1)
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	bCommand := []byte(fmt.Sprintf("\x1B*%c00", 33))
	bCommand[3] = byte(height % 256)
	bCommand[4] = byte(height / 256)
	data := []byte{0, 0, 0}
	for i := 0; i < (height/24 + 1); i++ {
		raw:=bCommand
		for j := 0; j < width; j++ {
			for k := 0; k < 24; k++ {
				if i*24+k < height {
					r, g, b, _ := img.At(j, (i*24 + k)).RGBA()
					if (r+g+b)/3/255 < 128 {
						data[k/8] += byte(128 >> uint(k%8))
					}
				}
			}
			raw = append(raw, data...)
			data = []byte{0, 0, 0}
		}
		e.WriteRaw(raw)
		e.Linefeed()

	}
	e.SetLineSpace(0)
}
