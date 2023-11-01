package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/flopp/go-findfont"
	"github.com/goki/freetype/truetype"
)

type app_config struct {
	EditWidget    *widget.Entry
	PreviewWidget *widget.RichText
	CurrentFile   fyne.URI
	SaveMenuItem  *fyne.MenuItem
}

// 创建UI函数，
func (app *app_config) makeUI() (*widget.Entry, *widget.RichText) {
	// 创建多行输入框widget
	edit := widget.NewMultiLineEntry()
	// 创建富文本框，for markdown
	preview := widget.NewRichTextFromMarkdown("")

	app.EditWidget = edit
	app.PreviewWidget = preview

	// 绑定到OnChanged, 绑定的内容为 fresh markdown
	edit.OnChanged = preview.ParseMarkdown

	return edit, preview
}

// 创建菜单函数
func (app *app_config) createMenuItems(window fyne.Window) {
	open_menu_item := fyne.NewMenuItem("Open", app.open_func(window))
	save_menu_item := fyne.NewMenuItem("Save", app.save_func(window))

	// 下面两行是默认先把save选项 disabled掉，等到save as之后， app.CurrentFile就会指向当前file
	// 再激活save as以保存到引用的file
	app.SaveMenuItem = save_menu_item
	app.SaveMenuItem.Disabled = true

	save_as_menu_item := fyne.NewMenuItem("Save as", app.save_as_func(window))

	file_menu := fyne.NewMenu("file", open_menu_item, save_menu_item, save_as_menu_item)

	main_menu := fyne.NewMainMenu(file_menu)

	window.SetMainMenu(main_menu)
}

var cfg app_config

// 处理中文显示乱码问题
func init() {
	fontPath, err := findfont.Find("SmileySans-Oblique.ttf")
	if err != nil {
		// panic(err)
		fmt.Println("Chinese Font Set Error!!!")
		return
	}

	fmt.Printf("Found 'SmileySans-Oblique.ttf' in '%s'\n", fontPath)

	// load the font with the freetype library
	// 原作者使用的ioutil.ReadFile已经弃用
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		// panic(err)
		fmt.Println("Font Read Failed:", err)
		return
	}

	_, err = truetype.Parse(fontData)
	if err != nil {
		// panic(err)
		fmt.Println("Parse Font Data Error:", err)
		return
	}
	os.Setenv("FYNE_FONT", fontPath)
}

func main() {

	// Create a Fyne App   创建App
	a := app.New()

	// Create a Window for the app  创建window
	w := a.NewWindow("Fyne-Markdown")

	// get the user interface  获取用户界面
	edit, preview := cfg.makeUI()
	// 创建菜单，
	cfg.createMenuItems(w)

	// set the content of the window  设置内容
	// 设置窗口窗体大小
	w.Resize(fyne.Size{Width: 960, Height: 540})
	w.SetContent(container.NewHSplit(edit, preview))

	// 设置windows居中显示
	w.CenterOnScreen()

	// show and run
	w.ShowAndRun()
}

// 文件过滤器，只显示.md .MD文件
var filter = storage.NewExtensionFileFilter([]string{".md", ".MD"})

// open file 打开文件的处理函数 返回类型是函数，是因为，上边new_item时，它需要的是函数进行绑定
func (app *app_config) open_func(w fyne.Window) func() {
	return func() {
		// dialog.NewFileOpen() 参数 1.回调函数 2.parent fyne.Window
		open_dialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				// 用户点击取消
				return
			}

			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			// 将data内容写入到 Entry中
			cfg.EditWidget.SetText(string(data))

			// 处理当前文件的引用
			app.CurrentFile = reader.URI()
			//
			w.SetTitle(w.Title() + "-" + reader.URI().Name())
			app.SaveMenuItem.Disabled = false

		}, w)
		// open_dialog.SetLocation()
		open_dialog.SetFilter(filter)
		open_dialog.Show()
	}
}

// 处理save逻辑
func (app *app_config) save_func(w fyne.Window) func() {
	return func() {
		// save逻辑不需要dialog
		// 如果currentFile为空则return
		if app.CurrentFile != nil {
			write, err := storage.Writer(app.CurrentFile)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			write.Write([]byte(app.EditWidget.Text))
			defer write.Close()
		}
	}
}

// 另存为处理函数  返回 一个函数
func (app *app_config) save_as_func(w fyne.Window) func() {
	return func() {
		save_dialog := dialog.NewFileSave(func(write fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if write == nil {
				// 用户点击取消
				return
			}

			// 检查用户的文件命名，如果不为指定命名方式则不保存
			// strings.HasPrefix(s,prefix) 函数用来检测字符串s是否以指定的前缀开头 两个字符串参数
			// strings.HasSuffix(s,suffix) 函数用来检测字符串s是否以指定的后缀结尾
			if !strings.HasSuffix(strings.ToLower(write.URI().String()), ".md") {
				dialog.ShowInformation("提示", "使用.md或.MD作为后缀", w) // 两个参数 title 和 message
				return
			}

			// 保存逻辑
			write.Write([]byte(cfg.EditWidget.Text))

			// URI是一个接口 打开的文件资源的引用
			app.CurrentFile = write.URI()

			// 函数结束时关闭写入
			defer write.Close()

			// 修改Title 为 原先的tile+ 保存的names
			w.SetTitle(w.Title() + "-" + write.URI().Name())

			app.SaveMenuItem.Disabled = false

		}, w)
		//设置默认保存名
		save_dialog.SetFileName("untitled.md")
		// 在open dialog 和 save dialog都设置filter
		save_dialog.SetFilter(filter)
		save_dialog.Show()
	}
}
