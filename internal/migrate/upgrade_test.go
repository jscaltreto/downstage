package migrate

import (
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
