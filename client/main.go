package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/marcusolsson/tui-go"
)

type MsgType uint

const (
	IncomingMsg MsgType = iota
	OutgoingMsg
)

type Msg struct {
	Text string
	Type MsgType
}

func main() {
	history := tui.NewVBox()
	msgChannel := make(chan Msg)

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		text := e.Text()

		history.Append(tui.NewHBox(
			tui.NewLabel(fmt.Sprintf("<%s", text)),
			tui.NewSpacer(),
		))
		msgChannel <- Msg{
			Type: OutgoingMsg,
			Text: text + "\n",
		}
		input.SetText("")
	})

	root := tui.NewHBox(chat)

	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalln("Fail to connect to server")
	}

	go func() {
		reader := bufio.NewReader(conn)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				log.Fatalln("Fail to connect to server")
				return
			}

			msgChannel <- Msg{
				Text: fmt.Sprintf(">%s", string(line)),
				Type: IncomingMsg,
			}
		}
	}()

	go func() {
		for {
			msg := <-msgChannel
			switch msg.Type {
			case IncomingMsg:
				history.Append(tui.NewHBox(
					tui.NewLabel(msg.Text),
					tui.NewSpacer(),
				))
				ui.Repaint()
			case OutgoingMsg:
				conn.Write([]byte(msg.Text))
			}
		}
	}()

	history.Append(tui.NewHBox(
		tui.NewLabel("(	( ͡° ͜ʖ ͡°)ﾉ⌐■-■   You can exit by pressing Esc key!    (◕ᴥ◕ʋ))"),
		tui.NewSpacer(),
	))

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
