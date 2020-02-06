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
	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}

	go updateServices(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(view1, 0, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Services"
		fmt.Fprintln(v, "loading data...")
	}

	return nil
}

func updateServices(g *gocui.Gui) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-doneChan:
			return
		case <-ticker.C:
			table := []string{"Name | Status"}
			services := getServices()

			for _, service := range services {
				table = append(table, fmt.Sprintf("%s | %s ", service.Name, service.Status))
			}

			g.Update(func(g *gocui.Gui) error {
				v, err := g.View(view1)
				if err != nil {
					return err
				}

				v.Clear()
				result := strings.Split(columnize.SimpleFormat(table), "\n")
				boldWhite.Fprintln(v, result[0])
				fmt.Fprintln(v, strings.Join(result[1:], "\n"))
				return nil
			})
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

func quit(g *gocui.Gui, v *gocui.View) error {
	close(doneChan)
	return gocui.ErrQuit
}

var serviceRegex = `^(\w+)\s+(\w+)\s+(.*)$`

var (
	boldWhite  = color.New(color.FgWhite, color.Bold)
	boldGreen  = color.New(color.FgGreen, color.Bold)
	boldYellow = color.New(color.FgYellow, color.Bold)
	boldRed    = color.New(color.FgRed, color.Bold)
	view1      = "v1"
	doneChan   = make(chan struct{})
)

type Service struct {
	Status string
	Name   string
}
