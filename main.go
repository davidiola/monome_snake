package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sgarcez/gomonome/monome"
)

const RIGHT int = 0
const LEFT int = 1
const UP int = 2
const DOWN int = 3

var direction int

type Tuple struct {
	R int32
	C int32
}

func gridDemo(port int32) *monome.Grid {
	g, err := monome.StartGrid(port)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			e := g.Read()
			if e == nil {
				break
			}
			go func(e monome.DeviceEvent) {
				switch e.(type) {
				case monome.ReadyEvent:
					fmt.Println("Activating Tilt Sensor")
					g.ActivateTilt()
				case monome.TiltEvent:
					te := e.(monome.TiltEvent)
					if te.X <= 105 {
						direction = RIGHT
					} else if te.X >= 140 {
						direction = LEFT
					} else if te.Y <= 115 {
						direction = UP
					} else if te.Y >= 147 {
						direction = DOWN
					}
					fmt.Println(direction)
				}
			}(e)
		}
	}()

	return g
}

func contains(snake []Tuple, t Tuple) bool {

	for _, snakePart := range snake {
		if snakePart == t {
			return true
		}
	}
	return false
}

func setNewPellet(g *monome.Grid, snake []Tuple) Tuple {

	//need snake location and ensure new pellet is not at any point of any part
	//of the snake

	//get two random numbers from 0-7 (random row + col)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	r := int32(r1.Intn(8))
	c := int32(r1.Intn(8))
	t := Tuple{r, c}

	for contains(snake, t) == true {
		s1 = rand.NewSource(time.Now().UnixNano())
		r1 = rand.New(s1)
		r = int32(r1.Intn(8))
		c = int32(r1.Intn(8))
		t = Tuple{r, c}
	}

	g.LedSet(c, r, 1)
	return t

}

func createSnake(g *monome.Grid) []Tuple {

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	r := int32(r1.Intn(8))
	c := int32(r1.Intn(8))

	//ensure snake does not overlap with pellet and does not spawn at a boundary
	for r == 0 || r == 7 || c == 0 || c == 7 {
		s1 = rand.NewSource(time.Now().UnixNano())
		r1 = rand.New(s1)
		r = int32(r1.Intn(8))
		c = int32(r1.Intn(8))
	}

	t := Tuple{r, c}
	g.LedSet(c, r, 1)
	return []Tuple{t}

}

func boundaryConditions(snake []Tuple) bool {
	if snake[0].R < 0 || snake[0].R > 7 || snake[0].C < 0 || snake[0].C > 7 {
		return true
	}
	for i, snakeBody := range snake {
		if i != 0 {
			if snakeBody.C == snake[0].C && snakeBody.R == snake[0].R {
				return true
			}
		}
	}

	return false
}

func main() {

	fps := 2 //frame rate for game
	score := 0

	s, err := monome.StartSerialOSC()
	if err != nil {
		panic(err)
	}

	var g *monome.Grid
	e, err := s.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(e, e.DeviceKind())

	//start with random direction
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	direction = r1.Intn(4)

	switch e.DeviceKind() {
	case "grid":
		g = gridDemo(e.Port)
	}

	//create snake with length 1 at random internal cells that aren't pellet
	snake := createSnake(g)
	fmt.Println(snake)
	//create food pellet at random cell
	pelletLoc := setNewPellet(g, snake)
	fmt.Println(pelletLoc)

	//check if snake hits itself
	for boundaryConditions(snake) == false {
		time.Sleep(time.Second / time.Duration(fps))
		//check if head of snake has hit pellet
		//next location?
		nextLocation := snake[0]
		switch direction {
		case LEFT:
			nextLocation.C -= 1
		case RIGHT:
			nextLocation.C += 1
		case DOWN:
			nextLocation.R += 1
		case UP:
			nextLocation.R -= 1
		}

		//front of slice is always head
		if nextLocation.R == pelletLoc.R && nextLocation.C == pelletLoc.C {
			appendToSnake := pelletLoc
			//fix this
			snake = append([]Tuple{appendToSnake}, snake...)
			g.LedSet(appendToSnake.C, appendToSnake.R, 1)
			//set new head to be the pelletLocation and move all previous locations up one
			//spawn a new pelletLocation
			pelletLoc = setNewPellet(g, snake)
			score++
		} else { //did not hit pellet, shift snake locations
			snake = append([]Tuple{nextLocation}, snake...)
			g.LedSet(nextLocation.C, nextLocation.R, 1)
			g.LedSet(snake[len(snake)-1].C, snake[len(snake)-1].R, 0)
			snake = snake[:len(snake)-1]
		}
	}

	fmt.Printf("Game Over! Score: %d\n", score)
	s.Close()
	waitOnSignal(g)
}

func waitOnSignal(g *monome.Grid) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	g.LedAll(false)

	<-done
	fmt.Println("exiting")
}
