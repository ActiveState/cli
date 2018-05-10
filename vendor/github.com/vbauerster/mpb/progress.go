package mpb

import (
	"container/heap"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/vbauerster/mpb/cwriter"
)

const (
	// default RefreshRate
	prr = 120 * time.Millisecond
	// default width
	pwidth = 80
	// default format
	pformat = "[=>-]"
)

// Progress represents the container that renders Progress bars
type Progress struct {
	wg           *sync.WaitGroup
	uwg          *sync.WaitGroup
	operateState chan func(*pState)
	done         chan struct{}
}

type (
	// progress state, which may contain several bars
	pState struct {
		bHeap           *priorityQueue
		shutdownPending []*Bar
		heapUpdated     bool
		zeroWait        bool
		idCounter       int
		width           int
		format          string
		rr              time.Duration
		cw              *cwriter.Writer
		ticker          *time.Ticker

		// following are provided by user
		uwg              *sync.WaitGroup
		cancel           <-chan struct{}
		shutdownNotifier chan struct{}
		interceptors     []func(io.Writer)
		waitBars         map[*Bar]*Bar
		debugOut         io.Writer
	}
	widthSyncer struct {
		// Public for easy testing
		Accumulator []chan int
		Distributor []chan int
	}
)

// New creates new Progress instance, which orchestrates bars rendering process.
// Accepts mpb.ProgressOption funcs for customization.
func New(options ...ProgressOption) *Progress {
	pq := make(priorityQueue, 0)
	heap.Init(&pq)
	s := &pState{
		bHeap:    &pq,
		width:    pwidth,
		format:   pformat,
		cw:       cwriter.New(os.Stdout),
		rr:       prr,
		ticker:   time.NewTicker(prr),
		waitBars: make(map[*Bar]*Bar),
		debugOut: ioutil.Discard,
	}

	for _, opt := range options {
		if opt != nil {
			opt(s)
		}
	}

	p := &Progress{
		uwg:          s.uwg,
		wg:           new(sync.WaitGroup),
		operateState: make(chan func(*pState)),
		done:         make(chan struct{}),
	}
	go p.serve(s)
	return p
}

// AddBar creates a new progress bar and adds to the container.
func (p *Progress) AddBar(total int64, options ...BarOption) *Bar {
	p.wg.Add(1)
	result := make(chan *Bar, 1)
	select {
	case p.operateState <- func(s *pState) {
		options = append(options, barWidth(s.width), barFormat(s.format))
		b := newBar(p.wg, s.idCounter, total, s.cancel, options...)
		if b.runningBar != nil {
			s.waitBars[b.runningBar] = b
		} else {
			heap.Push(s.bHeap, b)
			s.heapUpdated = true
		}
		s.idCounter++
		result <- b
	}:
		return <-result
	case <-p.done:
		p.wg.Done()
		return nil
	}
}

// Abort is only effective while bar progress is running,
// it means remove bar now without waiting for its completion.
// If bar is already completed, there is nothing to abort.
// If you need to remove bar after completion, use BarRemoveOnComplete BarOption.
func (p *Progress) Abort(b *Bar) {
	select {
	case p.operateState <- func(s *pState) {
		s.heapUpdated = heap.Remove(s.bHeap, b.index) != nil
		s.shutdownPending = append(s.shutdownPending, b)
	}:
	case <-p.done:
	}
}

// UpdateBarPriority provides a way to change bar's order position.
// Zero is highest priority, i.e. bar will be on top.
func (p *Progress) UpdateBarPriority(b *Bar, priority int) {
	select {
	case p.operateState <- func(s *pState) { s.bHeap.update(b, priority) }:
	case <-p.done:
	}
}

// BarCount returns bars count
func (p *Progress) BarCount() int {
	result := make(chan int, 1)
	select {
	case p.operateState <- func(s *pState) { result <- s.bHeap.Len() }:
		return <-result
	case <-p.done:
		return 0
	}
}

// Wait first waits for user provided *sync.WaitGroup, if any,
// then waits far all bars to complete and finally shutdowns master goroutine.
// After this method has been called, there is no way to reuse *Progress instance.
func (p *Progress) Wait() {
	if p.uwg != nil {
		p.uwg.Wait()
	}

	p.wg.Wait()

	select {
	case p.operateState <- func(s *pState) { s.zeroWait = true }:
		<-p.done
	case <-p.done:
	}
}

func newWidthSyncer(timeout <-chan struct{}, numBars, numColumn int) *widthSyncer {
	ws := &widthSyncer{
		Accumulator: make([]chan int, numColumn),
		Distributor: make([]chan int, numColumn),
	}
	for i := 0; i < numColumn; i++ {
		ws.Accumulator[i] = make(chan int, numBars)
		ws.Distributor[i] = make(chan int, numBars)
	}
	for i := 0; i < numColumn; i++ {
		go func(accumulator <-chan int, distributor chan<- int) {
			defer close(distributor)
			widths := make([]int, 0, numBars)
		loop:
			for {
				select {
				case w := <-accumulator:
					widths = append(widths, w)
					if len(widths) == numBars {
						break loop
					}
				case <-timeout:
					if len(widths) == 0 {
						return
					}
					break loop
				}
			}
			maxWidth := calcMax(widths)
			for i := 0; i < len(widths); i++ {
				distributor <- maxWidth
			}
		}(ws.Accumulator[i], ws.Distributor[i])
	}
	return ws
}

func (s *pState) writeAndFlush(tw, numP, numA int) (err error) {
	timeout := make(chan struct{})
	pSyncer := newWidthSyncer(timeout, s.bHeap.Len(), numP)
	aSyncer := newWidthSyncer(timeout, s.bHeap.Len(), numA)
	time.AfterFunc(s.rr-s.rr/12, func() {
		close(timeout)
	})

	for _, ch := range s.renderByPriority(tw, pSyncer, aSyncer) {
		bf := <-ch
		_, err = s.cw.ReadFrom(bf.reader)
		if !bf.bar.completed && bf.toComplete {
			bf.bar.completed = true
			if bf.bar.removeOnComplete {
				s.heapUpdated = heap.Remove(s.bHeap, bf.bar.index) != nil
			}
			if replacementBar, ok := s.waitBars[bf.bar]; ok {
				heap.Push(s.bHeap, replacementBar)
				s.heapUpdated = true
				delete(s.waitBars, bf.bar)
			}
			defer func() {
				s.shutdownPending = append(s.shutdownPending, bf.bar)
			}()
		}
	}

	for _, interceptor := range s.interceptors {
		interceptor(s.cw)
	}

	if e := s.cw.Flush(); err == nil {
		err = e
	}

	for i := len(s.shutdownPending) - 1; i >= 0; i-- {
		close(s.shutdownPending[i].shutdown)
		s.shutdownPending = s.shutdownPending[:i]
	}
	return
}

func (s *pState) renderByPriority(tw int, pSyncer, aSyncer *widthSyncer) []<-chan *bFrame {
	bff := make([]<-chan *bFrame, 0, s.bHeap.Len())
	for s.bHeap.Len() > 0 {
		b := heap.Pop(s.bHeap).(*Bar)
		defer heap.Push(s.bHeap, b)
		bff = append(bff, b.render(s.debugOut, tw, pSyncer, aSyncer))
	}
	return bff
}

func calcMax(slice []int) int {
	if len(slice) == 0 {
		return 0
	}

	max := slice[0]
	for i := 1; i < len(slice); i++ {
		if slice[i] > max {
			max = slice[i]
		}
	}
	return max
}
