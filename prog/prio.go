// Copyright 2015/2016 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/google/syzkaller/pkg/log"
)

// Calulation of call-to-call priorities.
// For a given pair of calls X and Y, the priority is our guess as to whether
// additional of call Y into a program containing call X is likely to give
// new coverage or not.
// The current algorithm has two components: static and dynamic.
// The static component is based on analysis of argument types. For example,
// if call X and call Y both accept fd[sock], then they are more likely to give
// new coverage together.
// The dynamic component is based on frequency of occurrence of a particular
// pair of syscalls in a single program in corpus. For example, if socket and
// connect frequently occur in programs together, we give higher priority to
// this pair of syscalls.
// Note: the current implementation is very basic, there is no theory behind any
// constants.

func (target *Target) CalculatePriorities(corpus []*Prog) [][]float32 {
	static := target.calcStaticPriorities()
	if len(corpus) != 0 {
		dynamic := target.calcDynamicPrio(corpus)
		for i, prios := range dynamic {
			for j, p := range prios {
				static[i][j] *= p
			}
		}
	}
	return static
}

func (target *Target) calcStaticPriorities() [][]float32 {
	uses := target.calcResourceUsage()
	prios := make([][]float32, len(target.Syscalls))
	for i := range prios {
		prios[i] = make([]float32, len(target.Syscalls))
	}
	var keys []string
	for key := range uses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		weights := make([]weights, 0, len(uses[key]))
		for _, weight := range uses[key] {
			weights = append(weights, weight)
		}
		sort.Slice(weights, func(i, j int) bool {
			return weights[i].call < weights[j].call
		})
		for _, w0 := range weights {
			for _, w1 := range weights {
				if w0.call == w1.call {
					// Self-priority is assigned below.
					continue
				}
				// The static priority is assigned based on the direction of arguments. A higher priority will be
				// assigned when c0 is a call that produces a resource and c1 a call that uses that resource.
				prios[w0.call][w1.call] += w0.inout*w1.in + 0.7*w0.inout*w1.inout
			}
		}
	}
	normalizePrio(prios)
	// The value assigned for self-priority (call wrt itself) have to be high, but not too high.
	for c0, pp := range prios {
		pp[c0] = 0.9
	}
	return prios
}

func (target *Target) calcResourceUsage() map[string]map[int]weights {
	uses := make(map[string]map[int]weights)
	ForeachType(target.Syscalls, func(t Type, ctx TypeCtx) {
		c := ctx.Meta
		switch a := t.(type) {
		case *ResourceType:
			if target.AuxResources[a.Desc.Name] {
				noteUsage(uses, c, 0.1, ctx.Dir, "res%v", a.Desc.Name)
			} else {
				str := "res"
				for i, k := range a.Desc.Kind {
					str += "-" + k
					w := 1.0
					if i < len(a.Desc.Kind)-1 {
						w = 0.2
					}
					noteUsage(uses, c, float32(w), ctx.Dir, str)
				}
			}
		case *PtrType:
			if _, ok := a.Elem.(*StructType); ok {
				noteUsage(uses, c, 1.0, ctx.Dir, "ptrto-%v", a.Elem.Name())
			}
			if _, ok := a.Elem.(*UnionType); ok {
				noteUsage(uses, c, 1.0, ctx.Dir, "ptrto-%v", a.Elem.Name())
			}
			if arr, ok := a.Elem.(*ArrayType); ok {
				noteUsage(uses, c, 1.0, ctx.Dir, "ptrto-%v", arr.Elem.Name())
			}
		case *BufferType:
			switch a.Kind {
			case BufferBlobRand, BufferBlobRange, BufferText:
			case BufferString:
				if a.SubKind != "" {
					noteUsage(uses, c, 0.2, ctx.Dir, fmt.Sprintf("str-%v", a.SubKind))
				}
			case BufferFilename:
				noteUsage(uses, c, 1.0, DirIn, "filename")
			default:
				panic("unknown buffer kind")
			}
		case *VmaType:
			noteUsage(uses, c, 0.5, ctx.Dir, "vma")
		case *IntType:
			switch a.Kind {
			case IntPlain, IntRange:
			default:
				panic("unknown int kind")
			}
		}
	})
	return uses
}

type weights struct {
	call  int
	in    float32
	inout float32
}

func noteUsage(uses map[string]map[int]weights, c *Syscall, weight float32, dir Dir, str string, args ...interface{}) {
	id := fmt.Sprintf(str, args...)
	if uses[id] == nil {
		uses[id] = make(map[int]weights)
	}
	callWeight := uses[id][c.ID]
	callWeight.call = c.ID
	if dir != DirOut {
		if weight > uses[id][c.ID].in {
			callWeight.in = weight
		}
	}
	if weight > uses[id][c.ID].inout {
		callWeight.inout = weight
	}
	uses[id][c.ID] = callWeight
}

func (target *Target) calcDynamicPrio(corpus []*Prog) [][]float32 {
	prios := make([][]float32, len(target.Syscalls))
	for i := range prios {
		prios[i] = make([]float32, len(target.Syscalls))
	}
	for _, p := range corpus {
		for idx0, c0 := range p.Calls {
			for _, c1 := range p.Calls[idx0+1:] {
				id0 := c0.Meta.ID
				id1 := c1.Meta.ID
				prios[id0][id1] += 1.0
				// prios[id0][id1] += float32(c1.Stat.LockStat)
			}
		}
	}
	normalizePrio(prios)
	return prios
}

// normalizePrio assigns some minimal priorities to calls with zero priority,
// and then normalizes priorities to 0.1..1 range.
func normalizePrio(prios [][]float32) {
	for _, prio := range prios {
		max := float32(0)
		min := float32(1e10)
		nzero := 0
		for _, p := range prio {
			if max < p {
				max = p
			}
			if p != 0 && min > p {
				min = p
			}
			if p == 0 {
				nzero++
			}
		}
		if nzero != 0 {
			min /= 2 * float32(nzero)
		}
		if min == max {
			max = 0
		}
		for i, p := range prio {
			if max == 0 {
				prio[i] = 1
				continue
			}
			if p == 0 {
				p = min
			}
			p = (p-min)/(max-min)*0.9 + 0.1
			if p > 1 {
				p = 1
			}
			prio[i] = p
		}
	}
}

// ChooseTable allows to do a weighted choice of a syscall for a given syscall
// based on call-to-call priorities and a set of enabled syscalls.
type ChoiceTable struct {
	target *Target
	runs   [][]int
	calls  []*Syscall
}

func (target *Target) BuildChoiceTable(corpus []*Prog, enabled map[*Syscall]bool) *ChoiceTable {
	if enabled == nil {
		enabled = make(map[*Syscall]bool)
		for _, c := range target.Syscalls {
			enabled[c] = true
		}
	}
	for call := range enabled {
		if call.Attrs.Disabled {
			delete(enabled, call)
		}
	}
	var enabledCalls []*Syscall
	for c := range enabled {
		enabledCalls = append(enabledCalls, c)
	}
	if len(enabledCalls) == 0 {
		panic("no syscalls enabled")
	}
	sort.Slice(enabledCalls, func(i, j int) bool {
		return enabledCalls[i].ID < enabledCalls[j].ID
	})
	for _, p := range corpus {
		for _, call := range p.Calls {
			if !enabled[call.Meta] {
				fmt.Printf("corpus contains disabled syscall %v", call.Meta.Name)
				panic("disabled syscall")
			}
		}
	}
	prios := target.CalculatePriorities(corpus)
	run := make([][]int, len(target.Syscalls))
	for i := range run {
		if !enabled[target.Syscalls[i]] {
			continue
		}
		run[i] = make([]int, len(target.Syscalls))
		sum := 0
		for j := range run[i] {
			if enabled[target.Syscalls[j]] {
				sum += int(prios[i][j] * 1000)
			}
			run[i][j] = sum
		}
	}
	return &ChoiceTable{target, run, enabledCalls}
}

func (ct *ChoiceTable) Enabled(call int) bool {
	return ct.runs[call] != nil
}

func (ct *ChoiceTable) choose(r *rand.Rand, bias int) int {
	if bias < 0 {
		bias = ct.calls[r.Intn(len(ct.calls))].ID
	}
	if !ct.Enabled(bias) {
		fmt.Printf("bias to disabled syscall %v", ct.target.Syscalls[bias].Name)
		panic("disabled syscall")
	}
	run := ct.runs[bias]
	x := r.Intn(run[len(run)-1]) + 1
	res := sort.SearchInts(run, x)
	if !ct.Enabled(res) {
		panic("selected disabled syscall")
	}
	return res
}

// prog, prio relational table

type ProgTemplate struct {
	syscallNum int
	prog       []*Syscall
}

func toProgTemplate(p []*Syscall) *ProgTemplate {
	snum := len(p)
	var tempProg []*Syscall
	for _, s := range p {
		for _, existS := range tempProg {
			if s.ID == existS.ID {
				break
			}
		}
		tempProg = append(tempProg, s)
	}
	result := &ProgTemplate{
		syscallNum: snum,
		prog:       tempProg,
	}
	return result
}

func (pt *ProgTemplate) containSyscall(s *Syscall) bool {
	for _, pts := range pt.prog {
		if s.ID == pts.ID {
			return true
		}
	}
	return false
}

func (pt *ProgTemplate) containProg(p []*Syscall) bool {
	for _, s := range p {
		found := false
		for _, origS := range pt.prog {
			if s.ID == origS.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (pt *ProgTemplate) equal(pt1 *ProgTemplate) bool {
	if pt.syscallNum != pt1.syscallNum {
		return false
	}
	for _, s := range pt.prog {
		if !pt1.containSyscall(s) {
			return false
		}
	}
	return true
}

func (pt *ProgTemplate) GenerateRndProgram(s *state, progLen int) *Prog {
	prog2Gen := make([]*Syscall, progLen)
	for i := len(prog2Gen); i < progLen; i++ {
		prog2Gen = append(prog2Gen, pt.prog[rand.Intn(len(pt.prog))])
	}
	progGenerated := make([]*Call, progLen)
	var r randGen
	for i := 0; i < len(prog2Gen); i++ {
		tempCalls := r.generateParticularCall(s, prog2Gen[i])
		progGenerated[i] = tempCalls[0]
	}
	result := &Prog{
		Target: s.target,
		Calls:  progGenerated,
	}
	return result
}

type RelationEdge struct {
	from  *ProgTemplate
	to    *ProgTemplate
	value int
}

type ProgTempRelationGraph struct {
	target *Target
	edges  []*RelationEdge
}

func (re *RelationEdge) sameSrcDst(from *ProgTemplate, to *ProgTemplate) bool {
	if re.from.equal(from) && re.to.equal(to) {
		return true
	}
	return false
}

func (ptrg *ProgTempRelationGraph) hasEdge(from *ProgTemplate, to *ProgTemplate) bool {
	for _, e := range ptrg.edges {
		if e.sameSrcDst(from, to) {
			return true
		}
	}
	return false
}

func (ptrg *ProgTempRelationGraph) addEdge(from *ProgTemplate, to *ProgTemplate, value int) {
	if ptrg.hasEdge(from, to) {
		log.Logf(1, "ProgTempRelationTable: edge already exists")
		return
	} else {
		newEdge := &RelationEdge{
			from:  from,
			to:    to,
			value: value,
		}
		ptrg.edges = append(ptrg.edges, newEdge)
	}
}

func (ptrg *ProgTempRelationGraph) getProgTemplatesSet() map[*ProgTemplate]bool {
	var result map[*ProgTemplate]bool
	for _, e := range ptrg.edges {
		if !result[e.from] {
			result[e.from] = true
		}
		if !result[e.to] {
			result[e.to] = true
		}
	}
	return result
}

func (ptrg *ProgTempRelationGraph) changeEdgeValue(f *ProgTemplate, t *ProgTemplate, newVal int) {
	if !ptrg.hasEdge(f, t) {
		log.Logf(1, "ProgTempRelationTable: edge does not exist")
		ptrg.addEdge(f, t, newVal)
	} else {
		for _, e := range ptrg.edges {
			if e.from.equal(f) && e.to.equal(t) {
				e.value = newVal
				break
			}
		}
	}
}

func (ptrg *ProgTempRelationGraph) getEdgeValue(f *ProgTemplate, t *ProgTemplate) int {
	if !ptrg.hasEdge(f, t) {
		// log.Logf(1, "ProgTempRelationTable: edge does not exist")
		return -1
	} else {
		for _, e := range ptrg.edges {
			if e.from.equal(f) && e.to.equal(t) {
				return e.value
			}
		}
		return -1
	}
}

func (ptrg *ProgTempRelationGraph) createOrIncreaseEdgeValue(f *ProgTemplate, t *ProgTemplate, incVal int) {
	if ptrg.hasEdge(f, t) {
		ptrg.changeEdgeValue(f, t, ptrg.getEdgeValue(f, t)+incVal)
	} else {
		ptrg.edges = append(ptrg.edges, &RelationEdge{
			from:  f,
			to:    t,
			value: incVal,
		})
	}
}

func (target *Target) BuildProgRelationGraph(enabled map[*Syscall]bool) *ProgTempRelationGraph {
	if enabled == nil {
		enabled = make(map[*Syscall]bool)
		for _, c := range target.Syscalls {
			enabled[c] = true
		}
	}

	var progTemplates []*ProgTemplate
	for _, c := range target.Syscalls {
		var prog []*Syscall
		prog = append(prog, c)
		callNum := 1
		currProgTemplate := &ProgTemplate{
			syscallNum: callNum,
			prog:       prog,
		}
		progTemplates = append(progTemplates, currProgTemplate)
	}

	var result *ProgTempRelationGraph
	var edges []*RelationEdge
	result = &ProgTempRelationGraph{
		target: target,
		edges:  edges,
	}
	return result
}

func (ptrg *ProgTempRelationGraph) getProgTemplates() []*ProgTemplate {

	progTemplatesSet := ptrg.getProgTemplatesSet()
	var progTemplates []*ProgTemplate
	for key := range progTemplatesSet {
		progTemplates = append(progTemplates, key)
	}
	return progTemplates
}

func (ptrg *ProgTempRelationGraph) getRelationTable() map[*ProgTemplate]map[*ProgTemplate]float32 {
	var result map[*ProgTemplate]map[*ProgTemplate]float32
	progTemplates := ptrg.getProgTemplates()
	for _, p1 := range progTemplates {
		for _, p2 := range progTemplates {
			if ptrg.hasEdge(p1, p2) {
				result[p1][p2] = float32(ptrg.getEdgeValue(p1, p2))
			} else {
				result[p1][p2] = 0
			}
		}
	}
	return result
}

func (ptrg *ProgTempRelationGraph) Update(progs []*Prog, times int) {
	progTemps := make([]*ProgTemplate, len(progs))
	progPrios := make([]int, len(progs))
	for _, p := range progs {
		programSyscalls := make([]*Syscall, len(p.Calls))
		for _, origC := range p.Calls {
			programSyscalls = append(programSyscalls, origC.Meta)
		}
		ptCreated := false
		for _, pt := range ptrg.getProgTemplates() {
			if pt.containProg(programSyscalls) {
				ptCreated = true
				progTemps = append(progTemps, pt)
				break
			}
		}
		if !ptCreated {
			newPt := toProgTemplate(programSyscalls)
			progTemps = append(progTemps, newPt)
		}
		progPrios = append(progPrios, p.Prio)
	}
	for i := 0; i < len(progTemps); i++ {
		for j := i + 1; j < len(progTemps); j++ {
			if progPrios[i] < progPrios[j] {
				ptrg.createOrIncreaseEdgeValue(progTemps[j], progTemps[i], times)
			} else if progPrios[i] > progPrios[j] {
				ptrg.createOrIncreaseEdgeValue(progTemps[i], progTemps[j], times)
			} else {
				ptrg.createOrIncreaseEdgeValue(progTemps[j], progTemps[i], times/2)
				ptrg.createOrIncreaseEdgeValue(progTemps[i], progTemps[j], times/2)
			}
		}
	}
}

func (ptrg *ProgTempRelationGraph) ChoosePrograms(minVal float32, taskLen int) []*ProgTemplate {
	progTemplates := ptrg.getProgTemplates()
	var max float32
	for i := 0; i < len(progTemplates); i++ {
		for j := 0; j < len(progTemplates); j++ {
			tempVal := float32(ptrg.getEdgeValue(progTemplates[i], progTemplates[j]))
			if tempVal > max {
				max = tempVal
			}
		}
	}
	var usedTable map[*ProgTemplate]map[*ProgTemplate]float32
	for i := 0; i < len(progTemplates); i++ {
		for j := 0; j < len(progTemplates); j++ {
			tempVal := float32(ptrg.getEdgeValue(progTemplates[i], progTemplates[j]))
			usedTable[progTemplates[i]][progTemplates[j]] = tempVal / max

		}
	}

	result := make([]*ProgTemplate, taskLen)

	for i := 0; i < taskLen; i++ {
		maxMinRandNum := rand.Intn(10)
			if maxMinRandNum >=0 && maxMinRandNum < 6 {
				result[i] = ptrg.computeChooseProgValueMax(usedTable, progTemplates, result, i)
			} else if maxMinRandNum >= 6 && maxMinRandNum < 8 {
				result[i] = ptrg.computeChooseProgValueMin(usedTable, progTemplates, result, i)
			} else {
				result[i] = ptrg.computeChooseProgValueRandom(progTemplates)
			}
		}
	progs := result
	return progs
}

func (ptrg *ProgTempRelationGraph) computeChooseProgValueMax(usedTable map[*ProgTemplate]map[*ProgTemplate]float32, 
	progTemplates []*ProgTemplate, 
	currResult []*ProgTemplate, 
	currLevel int) *ProgTemplate {
	var max float32
	var pt *ProgTemplate
	max = 0.0
	for i := 0; i < len(progTemplates); i++ {
		var val float32
		val = 0.0
		for l := 0; l < currLevel-1; l++ {
			val += usedTable[progTemplates[i]][currResult[l]]
		}
		if val > max {
			max = val
			pt = progTemplates[i]
		}
	}
	return pt
}

func (ptrg *ProgTempRelationGraph) computeChooseProgValueRandom(progTemplates []*ProgTemplate) *ProgTemplate {
	return progTemplates[rand.Intn(len(progTemplates))]
}

func (ptrg *ProgTempRelationGraph) computeChooseProgValueMin(usedTable map[*ProgTemplate]map[*ProgTemplate]float32, progTemplates []*ProgTemplate, currResult []*ProgTemplate, currLevel int) *ProgTemplate {
	var min float32
	var pt *ProgTemplate
	min = 0.0
	for i := 0; i < len(progTemplates); i++ {
		var val float32
		val = 0.0
		for l := 0; l < currLevel-1; l++ {
			val += usedTable[progTemplates[i]][currResult[l]]
		}
		if val < min {
			min = val
			pt = progTemplates[i]
		}
	}
	return pt
}
