package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Version holds the current project version
const Version = "0.0.1-dev1"

const filename = "gotuido.json"

var (
	// Initialize application
	app = tview.NewApplication()

	typingMode bool
	allTasks   Tasks
	path       string
)

type Task struct {
	ID          int64  `json:"id"`
	Theme       string `json:"theme"`
	Description string `json:"description"`
	Stage       int    `json:"stage"`
	Weight      int64  `json:"weight"`
}

type Tasks []Task

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	} else if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
	} else if runtime.GOOS == "darwin" {
		home := os.Getenv("HOME")
		if home != "" {
			return home
		}
	}
	return os.Getenv("HOME")
}

// ReadTasks read json from file and populate t
func (t *Tasks) ReadTasks(path string) error {
	// if path empty then it's users homedir with default filename
	if path != "" {
		path = userHomeDir() + filename
	} else {
		path += filename
	}

	// create file if not exist or for the first time
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return t.WriteTasks(path)
	} else {
		byteArray, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		err = json.Unmarshal(byteArray, t)
		return err
	}
}

// WriteTasks write new json to file
func (t *Tasks) WriteTasks(path string) error {
	// if path empty then it's users homedir with default filename
	if path != "" {
		path = userHomeDir() + filename
	} else {
		path += filename
	}

	byteArray, err := json.MarshalIndent(t, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, byteArray, 0644)

	return nil
}

func StagePopulation(tasks Tasks, stage int, result tview.Primitive) {
	result.(*tview.List).Clear()
	for _, t := range tasks {
		if t.Stage == stage {
			result.(*tview.List).AddItem(t.Theme, t.Description, '*', nil)
		}
	}
}

func Submit(addTaskForm *tview.Form, stageA *tview.List, grid *tview.Grid) {

	// workaround if first task creation
	l := 0

	if len(allTasks) > 0 {
		l = len(allTasks) - 1
	}

	// workaround if first task creation
	if l == 0 {
		allTasks = append(allTasks, Task{
			ID:          0,
			Theme:       addTaskForm.GetFormItem(0).(*tview.InputField).GetText(),
			Description: addTaskForm.GetFormItem(1).(*tview.InputField).GetText(),
			Weight:      0,
			Stage:       0,
		})
	} else {
		allTasks = append(allTasks, Task{
			ID:          allTasks[l].ID + 1,
			Theme:       addTaskForm.GetFormItem(0).(*tview.InputField).GetText(),
			Description: addTaskForm.GetFormItem(1).(*tview.InputField).GetText(),
			Weight:      allTasks[l].ID + 1,
			Stage:       0,
		})
	}
	err := allTasks.WriteTasks(path)
	if err != nil {
		fmt.Println(err)
	}
	StagePopulation(allTasks, 0, stageA)
	typingMode = false
	app.SetRoot(grid, true).EnableMouse(true).SetFocus(grid)
}

// create newHeadPrimitive processor
func newHeadPrimitive(text string) tview.Primitive {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}

// Returns a new primitive which puts the provided primitive in the center and
// sets its size to the given width and height.
func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func quit(buttonLabel string, grid *tview.Grid) {
	if buttonLabel == "Quit" {
		// Exit the application
		app.Stop()
	} else if buttonLabel == "Cancel" {
		app.SetRoot(grid, true).EnableMouse(true).SetFocus(grid)
	}
}

func input(stages []string, stageA, stageB, stageC *tview.List, grid *tview.Grid, finderFocus tview.Primitive, event *tcell.EventKey) *tcell.EventKey {
	// Anything handled here will be executed on the main thread
	switch event.Key() {
	// Press Ctrl+A to create new task
	case tcell.KeyCtrlA:
		typingMode = true
		addTaskForm := tview.NewForm().
			AddInputField("Please enter task theme: ", "", 50, nil, nil).
			AddInputField("Please enter task description: ", "", 50, nil, nil).
			SetLabelColor(12) // ColorBlue
		addTaskForm.SetButtonsAlign(tview.AlignCenter).
			SetBorder(true).
			SetTitle("Create new task").
			SetTitleAlign(tview.AlignCenter)

		addTaskForm.AddButton("Submit", func() {
			Submit(addTaskForm, stageA, grid)
		})

		app.SetRoot(modal(addTaskForm, 70, 10), true).EnableMouse(true).SetFocus(addTaskForm)

		return nil
	// Ctrl+B to remove task
	case tcell.KeyCtrlB:
		finderFocus = app.GetFocus()

		// Remove task from allTasks
		n := 0
		for _, v := range allTasks {
			theme, desc := finderFocus.(*tview.List).GetItemText(finderFocus.(*tview.List).GetCurrentItem())
			if v.Theme != theme && v.Description != desc {
				allTasks[n] = v
				n++
			}
		}
		allTasks = allTasks[:n]

		// Remove task in TUI
		finderFocus.(*tview.List).RemoveItem(finderFocus.(*tview.List).GetCurrentItem())

		// write update to file
		err := allTasks.WriteTasks(path)
		if err != nil {
			fmt.Println(err)
		}

		//reDraw all stages
		StagePopulation(allTasks, 0, stageA)
		StagePopulation(allTasks, 1, stageB)
		StagePopulation(allTasks, 2, stageC)

		app.SetFocus(finderFocus)

	// WASD to move task to right, left, up and down
	case tcell.KeyRune:
		if typingMode {
			return event
		}
		switch event.Rune() {
		// d - to move right
		case 'd':
			finderFocus = app.GetFocus()
			theme, desc := finderFocus.(*tview.List).GetItemText(finderFocus.(*tview.List).GetCurrentItem())
			for i, t := range allTasks {
				if t.Theme == theme && t.Description == desc && allTasks[i].Stage != len(stages)-1 {
					allTasks[i].Stage += 1
					switch allTasks[i].Stage {
					case 0:
						finderFocus = stageA
					case 1:
						finderFocus = stageB
					case 2:
						finderFocus = stageC
					}
					break
				}
			}
			// write update to file
			err := allTasks.WriteTasks(path)
			if err != nil {
				fmt.Println(err)
			}
			//reDraw all stages
			StagePopulation(allTasks, 0, stageA)
			StagePopulation(allTasks, 1, stageB)
			StagePopulation(allTasks, 2, stageC)

			// focus on moved item
			app.SetFocus(finderFocus)
			finderFocus.(*tview.List).SetCurrentItem(finderFocus.(*tview.List).GetItemCount() - 1)

		// a - to move left
		case 'a':
			finderFocus = app.GetFocus()
			theme, _ := finderFocus.(*tview.List).GetItemText(finderFocus.(*tview.List).GetCurrentItem())
			for i, t := range allTasks {
				if t.Theme == theme && allTasks[i].Stage != 0 {
					allTasks[i].Stage -= 1
					switch allTasks[i].Stage {
					case 0:
						finderFocus = stageA
					case 1:
						finderFocus = stageB
					case 2:
						finderFocus = stageC
					}
					break
				}
			}
			// write update to file
			err := allTasks.WriteTasks(path)
			if err != nil {
				fmt.Println(err)
			}
			//reDraw all stages
			StagePopulation(allTasks, 0, stageA)
			StagePopulation(allTasks, 1, stageB)
			StagePopulation(allTasks, 2, stageC)

			// focus on moved item
			app.SetFocus(finderFocus)
			finderFocus.(*tview.List).SetCurrentItem(finderFocus.(*tview.List).GetItemCount() - 1)
		}
	// Press Esc to exit from the app
	case tcell.KeyEsc:
		// Create a quit modal dialog
		m := tview.NewModal().
			SetText("Do you want to quit the application?").
			AddButtons([]string{"Quit", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				quit(buttonLabel, grid)
			})

		// Display and focus the dialog
		app.SetRoot(m, true).SetFocus(m)
		return nil
	}
	return event
}

// run execute main logic
func run(path string, stages []string) (err error) {
	err = allTasks.ReadTasks(path)
	if err != nil {
		return err
	}

	finderFocus := app.GetFocus()

	// declare heads
	header := newHeadPrimitive("Simple golang todo App")
	footer := newHeadPrimitive("Created with love to console")
	stageAHead := newHeadPrimitive(stages[0])
	stageBHead := newHeadPrimitive(stages[1])
	stageCHead := newHeadPrimitive(stages[2])
	//declare stages and populate them
	stageA := tview.NewList()
	stageB := tview.NewList()
	stageC := tview.NewList()

	StagePopulation(allTasks, 0, stageA)
	StagePopulation(allTasks, 1, stageB)
	StagePopulation(allTasks, 2, stageC)

	// main grid and header with footer
	grid := tview.NewGrid().
		SetRows(1, 1, 0, 1).
		SetColumns(40, 40, 40, 0).
		SetBorders(true).
		AddItem(header, 0, 0, 1, 3, 0, 0, false).
		AddItem(footer, 3, 0, 1, 3, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(stageAHead, 1, 0, 1, 1, 0, 100, false).
		AddItem(stageA, 2, 0, 1, 1, 0, 100, true).
		AddItem(stageBHead, 1, 1, 1, 1, 0, 100, false).
		AddItem(stageB, 2, 1, 1, 1, 0, 100, false).
		AddItem(stageCHead, 1, 2, 1, 1, 0, 100, false).
		AddItem(stageC, 2, 2, 1, 1, 0, 100, false)

	// Set the grid as the application root and enable mouse
	app.SetRoot(grid, true).EnableMouse(true)

	// Declare all hotkeys
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return input(stages, stageA, stageB, stageC, grid, finderFocus, event)
	})

	// Run the application
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func main() {
	fmt.Println(Version)

	typingMode = false
	path = ""
	stages := []string{"To Do", "In Progress", "Done"}

	err := run(path, stages)
	if err != nil {
		fmt.Println(err)
	}
}
