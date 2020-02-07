package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/ryanuber/columnize"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		panic(err)
	}

	go updateServices(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding(view1, gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding(view1, gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}

	if err := g.SetKeybinding(view1, gocui.KeyEnter, gocui.ModNone, switchPage); err != nil {
		return err
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(view1, 0, 0, 15, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.Highlight = true
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack

		fmt.Fprintln(v, PageServices)
		fmt.Fprintln(v, PageProcesses)
	}

	if v, err := g.SetView(view2, 15, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintln(v, loadingMessage)
	}

	if _, err := g.SetCurrentView(view1); err != nil {
		return err
	}

	return nil
}

func switchPage(g *gocui.Gui, v *gocui.View) error {
	var title string
	var err error

	_, cy := v.Cursor()
	if title, err = v.Line(cy); err != nil {
		title = ""
	}

	doneChan <- true

	switch title {
	case PageServices:
		go updateServices(g)
	case PageProcesses:
		go updateProcesses(g)
	}

	return nil
}

func updateServicesOnce(g *gocui.Gui) {
	var (
		table    = []string{"Name | Status"}
		services = getServices()
	)

	for _, service := range services {
		table = append(table, fmt.Sprintf("%s | %s ", service.Name, service.Status))
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(view2)
		if err != nil {
			return err
		}

		v.Clear()
		v.Title = PageServices
		result := strings.Split(columnize.SimpleFormat(table), "\n")
		boldWhite.Fprintln(v, result[0])
		fmt.Fprintln(v, strings.Join(result[1:], "\n"))
		return nil
	})
}

func updateServices(g *gocui.Gui) {
	ticker := time.NewTicker(5 * time.Second)

	updateServicesOnce(g)

	for {
		select {
		case <-doneChan:
			return
		case <-ticker.C:
			updateServicesOnce(g)
		}
	}
}

func getServices() []Service {
	// get services
	output, err := exec.Command("powershell.exe", "-c", "Get-Service | Format-Table -Auto").CombinedOutput()
	if err != nil {
		panic(err)
	}

	// Status      Name      DisplayName
	// ------      ----      -----------
	results := strings.Split(string(output), "\r\n")[3:]
	regex := regexp.MustCompile(serviceRegex)
	var services []Service

	for _, result := range results {
		matches := regex.FindStringSubmatch(result)

		if len(matches) != 4 {
			continue
		}

		//TODO: filter for only vcap services
		//if strings.TrimSpace(matches[3]) != "WalletService" {
		//	continue
		//}

		services = append(services, Service{
			Status: strings.TrimSpace(matches[1]),
			Name:   strings.TrimSpace(matches[2]),
		})
	}

	return services
}

func getProcesses() []Process {
	// get services
	output, err := exec.Command("powershell.exe", "-c", "Get-Process | Format-Table ID,Name").CombinedOutput()
	if err != nil {
		panic(err)
	}

	// Status      Name      DisplayName
	// ------      ----      -----------
	results := strings.Split(string(output), "\r\n")[3:]
	regex := regexp.MustCompile(processRegex)
	var processes []Process

	for _, result := range results {
		matches := regex.FindStringSubmatch(result)

		if len(matches) != 3 {
			continue
		}

		//TODO: filter for only vcap processes
		//if strings.TrimSpace(matches[3]) != "WalletService" {
		//	continue
		//}

		processes = append(processes, Process{
			ID:   strings.TrimSpace(matches[1]),
			Name: strings.TrimSpace(matches[2]),
		})
	}

	return processes
}

func updateProcesses(g *gocui.Gui) {
	ticker := time.NewTicker(5 * time.Second)

	updateProcessesOnce(g)

	for {
		select {
		case <-doneChan:
			return
		case <-ticker.C:
			updateProcessesOnce(g)
		}
	}
}

func updateProcessesOnce(g *gocui.Gui) {
	var (
		table    = []string{"ID | Name"}
		processes = getProcesses()
	)

	for _, process := range processes {
		table = append(table, fmt.Sprintf("%s | %s ", process.ID, process.Name))
	}

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(view2)
		if err != nil {
			return err
		}

		v.Clear()
		v.Title = PageProcesses
		result := strings.Split(columnize.SimpleFormat(table), "\n")
		boldWhite.Fprintln(v, result[0])
		fmt.Fprintln(v, strings.Join(result[1:], "\n"))
		return nil
	})
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	doneChan <- true
	return gocui.ErrQuit
}

var (
	serviceRegex   = `^(\w+)\s+(\w+)\s+(.*)$`
	processRegex   = `(\w+)\s+(\w+)`
	view1          = "v1"
	view2          = "v2"
	doneChan       = make(chan bool)
	boldWhite      = color.New(color.FgWhite, color.Bold)
	boldGreen      = color.New(color.FgGreen, color.Bold)
	boldYellow     = color.New(color.FgYellow, color.Bold)
	boldRed        = color.New(color.FgRed, color.Bold)
	PageServices   = "Services"
	PageProcesses  = "Processes"
	loadingMessage = "loading data..."
)

type Service struct {
	Status string
	Name   string
}

type Process struct {
	ID   string
	Name string
}
