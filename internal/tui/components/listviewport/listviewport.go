package listviewport

import (
	"time"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/dlvhdr/gh-dash/v4/internal/tui/constants"
	"github.com/dlvhdr/gh-dash/v4/internal/tui/context"
	"github.com/dlvhdr/gh-dash/v4/internal/utils"
)

type Model struct {
	ctx             context.ProgramContext
	viewport        viewport.Model
	topBoundId      int
	bottomBoundId   int
	currId          int
	yOffset         int
	ListItemHeight  int
	itemHeights     []int
	NumCurrentItems int
	NumTotalItems   int
	LastUpdated     time.Time
	CreatedAt       time.Time
	ItemTypeLabel   string
}

func NewModel(
	ctx context.ProgramContext,
	dimensions constants.Dimensions,
	lastUpdated time.Time,
	createdAt time.Time,
	itemTypeLabel string,
	numItems, listItemHeight int,
) Model {
	model := Model{
		ctx:             ctx,
		NumCurrentItems: numItems,
		ListItemHeight:  listItemHeight,
		currId:          0,
		viewport: viewport.New(
			viewport.WithWidth(dimensions.Width),
			viewport.WithHeight(dimensions.Height),
		),
		topBoundId:    0,
		ItemTypeLabel: itemTypeLabel,
		LastUpdated:   lastUpdated,
		CreatedAt:     createdAt,
	}
	model.bottomBoundId = utils.Min(
		model.NumCurrentItems-1,
		model.getNumPrsPerPage()-1,
	)
	return model
}

func (m *Model) SetNumItems(numItems int) {
	m.NumCurrentItems = numItems
	m.bottomBoundId = utils.Min(m.NumCurrentItems-1, m.getNumPrsPerPage()-1)
	m.currId = utils.Max(0, utils.Min(m.currId, m.NumCurrentItems-1))
}

// SetItemHeights records the rendered height of each logical selectable item.
// Decorations such as group headers can therefore take space without becoming
// selectable items themselves.
func (m *Model) SetItemHeights(heights []int) {
	m.itemHeights = append(m.itemHeights[:0], heights...)
	m.ensureVisible()
}

func (m *Model) SetTotalItems(total int) {
	m.NumTotalItems = total
}

func (m *Model) SetItemHeight(height int) {
	m.ListItemHeight = height
}

func (m *Model) SyncViewPort(content string) {
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(m.yOffset)
}

func (m *Model) getNumPrsPerPage() int {
	if m.ListItemHeight == 0 {
		return 0
	}
	return m.viewport.Height() / m.ListItemHeight
}

func (m *Model) ResetCurrItem() {
	m.currId = 0
	m.yOffset = 0
	m.viewport.GotoTop()
}

func (m *Model) GetCurrItem() int {
	return m.currId
}

func (m *Model) NextItem() int {
	newId := utils.Min(m.currId+1, m.NumCurrentItems-1)
	newId = utils.Max(newId, 0)
	m.currId = newId
	m.ensureVisible()
	return m.currId
}

func (m *Model) PrevItem() int {
	m.currId = utils.Max(m.currId-1, 0)
	m.ensureVisible()
	return m.currId
}

func (m *Model) SetCurrItem(id int) int {
	m.currId = utils.Max(0, utils.Min(id, m.NumCurrentItems-1))
	m.ensureVisible()
	return m.currId
}

func (m *Model) FirstItem() int {
	m.currId = 0
	m.yOffset = 0
	m.viewport.GotoTop()
	return m.currId
}

func (m *Model) LastItem() int {
	m.currId = m.NumCurrentItems - 1
	m.yOffset = max(0, m.itemStart(m.NumCurrentItems)-m.viewport.Height())
	m.viewport.GotoBottom()
	return m.currId
}

func (m *Model) itemHeight(id int) int {
	if id >= 0 && id < len(m.itemHeights) && m.itemHeights[id] > 0 {
		return m.itemHeights[id]
	}
	return m.ListItemHeight
}

func (m *Model) itemStart(id int) int {
	start := 0
	for i := 0; i < id; i++ {
		start += m.itemHeight(i)
	}
	return start
}

func (m *Model) ensureVisible() {
	if m.NumCurrentItems == 0 {
		m.yOffset = 0
		m.viewport.GotoTop()
		return
	}
	start := m.itemStart(m.currId)
	end := start + m.itemHeight(m.currId)
	top := m.yOffset
	bottom := top + m.viewport.Height()
	if start < top {
		m.yOffset = start
		m.viewport.SetYOffset(start)
	} else if end > bottom {
		m.yOffset = max(0, end-m.viewport.Height())
		m.viewport.SetYOffset(m.yOffset)
	}
	m.updateBounds()
}

func (m *Model) updateBounds() {
	top := m.yOffset
	bottom := top + m.viewport.Height()
	m.topBoundId = 0
	m.bottomBoundId = max(0, m.NumCurrentItems-1)
	for i := 0; i < m.NumCurrentItems; i++ {
		start := m.itemStart(i)
		end := start + m.itemHeight(i)
		if end > top {
			m.topBoundId = i
			break
		}
	}
	for i := m.topBoundId; i < m.NumCurrentItems; i++ {
		if m.itemStart(i) >= bottom {
			m.bottomBoundId = max(m.topBoundId, i-1)
			return
		}
	}
}

func (m *Model) SetDimensions(dimensions constants.Dimensions) {
	m.viewport.SetHeight(max(0, dimensions.Height))
	m.viewport.SetWidth(max(0, dimensions.Width))
}

func (m *Model) View() string {
	viewport := m.viewport.View()
	return lipgloss.NewStyle().
		Width(m.viewport.Width()).
		MaxWidth(m.viewport.Width()).
		Render(
			viewport,
		)
}

func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = *ctx
}
