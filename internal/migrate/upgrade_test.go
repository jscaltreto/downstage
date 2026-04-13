package migrate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpgradeV1ToV2_MigratesFrontmatterAndDramatisPersonae(t *testing.T) {
	source := `Title: Hamlet
Author: William Shakespeare
Date: 1603

# Dramatis Personae

HAMLET — Prince of Denmark
HORATIO
[HORATIO/HOR]

# Hamlet

## ACT I

### SCENE 1

HAMLET
To be.`

	upgraded, changed := UpgradeV1ToV2(source)
	assert.True(t, changed)
	assert.Equal(t, `# Hamlet
Author: William Shakespeare
Date: 1603

## Dramatis Personae

HAMLET - Prince of Denmark
HORATIO/HOR

## ACT I

### SCENE 1

HAMLET
To be.`, upgraded)
}

func TestUpgradeV1ToV2_CreatesHeadingWhenMissing(t *testing.T) {
	source := `Title: Untitled Example
Author: Someone

ALICE
Hello.`

	upgraded, changed := UpgradeV1ToV2(source)
	assert.True(t, changed)
	assert.Equal(t, `# Untitled Example
Author: Someone

ALICE
Hello.`, upgraded)
}

func TestUpgradeV1ToV2_LeavesV2DocumentAlone(t *testing.T) {
	source := `# Play
Author: Someone

## ACT I

### SCENE 1

ALICE
Hello.`

	upgraded, changed := UpgradeV1ToV2(source)
	assert.False(t, changed)
	assert.Equal(t, source, upgraded)
}

func TestUpgradeV1ToV2_PromotesDramatisPersonaeSubheadings(t *testing.T) {
	source := `Title: Play
Author: Me

# Dramatis Personae

ALICE
## The Servants
BOB

# Play

ALICE
Hi.`

	upgraded, changed := UpgradeV1ToV2(source)
	assert.True(t, changed)
	assert.Contains(t, upgraded, "## Dramatis Personae")
	assert.Contains(t, upgraded, "### The Servants")
}

func TestUpgradeV1ToV2_PreservesOrphanedAlias(t *testing.T) {
	source := `Title: Play

# Dramatis Personae

HAMLET — Prince of Denmark
[GHOST/SPECTRE]

# Play`

	upgraded, _ := UpgradeV1ToV2(source)
	// The orphan alias has no matching base entry, so the migrator emits it as
	// an inline alias rather than dropping it silently.
	assert.Contains(t, upgraded, "GHOST/SPECTRE")
}

func TestUpgradeV1ToV2_PreservesCustomKeyOrderDeterministically(t *testing.T) {
	source := `Title: Play
Author: Me
Producer: Studio
Dramaturg: Friend
Music: Composer

# Play`

	first, _ := UpgradeV1ToV2(source)
	for i := 0; i < 20; i++ {
		again, _ := UpgradeV1ToV2(source)
		if again != first {
			t.Fatalf("migration is nondeterministic:\nfirst:\n%s\nlater:\n%s", first, again)
		}
	}

	// Author is conventional (emitted first); the remaining custom keys must
	// come out in their source order.
	producerIdx := strings.Index(first, "Producer:")
	dramaturgIdx := strings.Index(first, "Dramaturg:")
	musicIdx := strings.Index(first, "Music:")
	if producerIdx < 0 || dramaturgIdx < 0 || musicIdx < 0 {
		t.Fatalf("missing custom key in output:\n%s", first)
	}
	if !(producerIdx < dramaturgIdx && dramaturgIdx < musicIdx) {
		t.Fatalf("custom keys not in source order:\n%s", first)
	}
}

func TestUpgradeV1ToV2_KeepsLeadingBlockComments(t *testing.T) {
	source := `/* Notes about this script. */

Title: Play
Author: Me

HAMLET
Hi.`

	upgraded, changed := UpgradeV1ToV2(source)
	// Leading block comments look like body content to the migrator, so it
	// leaves the file alone rather than guessing where the metadata sits.
	assert.False(t, changed)
	assert.Equal(t, source, upgraded)
}
