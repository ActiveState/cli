package renderers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBulletTree(t *testing.T) {
	assert.Equal(t, "├─ one\n├─ two \n│  wrapped\n└─ three",
		NewBulletList("", BulletTree, []string{"one", "two wrapped", "three"}).str(13))
}

func TestBulletTreeDisabled(t *testing.T) {
	assert.Equal(t, "[DISABLED]├─[/RESET] one\n[DISABLED]├─[/RESET] two \n[DISABLED]│ [/RESET] wrapped\n[DISABLED]└─[/RESET] three",
		NewBulletList("", BulletTreeDisabled, []string{"one", "two wrapped", "three"}).str(13))
}

func TestHeadedBulletTree(t *testing.T) {
	assert.Equal(t, "one\n├─ two \n│  wrapped\n└─ three",
		NewBulletList("", HeadedBulletTree, []string{"one", "two wrapped", "three"}).str(13))
}

func TestUnwrappedLink(t *testing.T) {
	assert.Equal(t, "├─ one\n└─ https://host:port/path#anchor",
		NewBulletList("", BulletTree, []string{"one", "https://host:port/path#anchor"}).str(10))
}

func TestIndented(t *testing.T) {
	// Note: use max width of 17 instead of 15 to account for extra continuation indent (2+2).
	assert.Equal(t, "  ├─ one\n  ├─ two \n  │  wrapped\n  └─ three",
		NewBulletList("  ", BulletTree, []string{"one", "two wrapped", "three"}).str(17))
}
