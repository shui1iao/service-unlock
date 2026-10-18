// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"testing"
)

func TestJCOMRemovedFromCatalogueRequestsParserAndAllGroups(t *testing.T) {
	id := "MediaUnlockTest_JCOM_ON_DEMAND"
	for _, c := range Catalogue() {
		if c.ID == id {
			t.Fatal("removed ID registered")
		}
	}
	for r, ids := range groupIDs {
		for _, x := range ids {
			if x == id {
				t.Fatal("removed expected ID", r)
			}
		}
	}
	if _, ok := requestCatalogue[id]; ok {
		t.Fatal("removed request retained")
	}
	if _, ok := singleParsers[id]; ok {
		t.Fatal("removed parser retained")
	}
	client, n := scripted(t, nil, 0)
	execute(context.Background(), client, id)
	if *n != 0 {
		t.Fatal("removed URL contacted")
	}
}
