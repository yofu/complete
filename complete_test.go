package complete

import (
	"fmt"
	"testing"
)

func TestComplete(t *testing.T) {
	var c *Complete

	// 'stress column l qx
	c, _ = Compile("'stress $ETYPE $PERIOD $NAME",
		map[string][]string{
			"ETYPE":  []string{"column", "girder", "brace", "wall", "wbrace", "slab", "sbrace"},
			"PERIOD": []string{"l", "x", "y"},
			"NAME":   []string{"n", "qx", "qy", "mz", "mx", "my"},
		})
	for _, s := range c.Complete("'stress column l q") {
		fmt.Println(s)
	}

	// :vim readme.txt
	c, _ = Compile(":vim %g", nil)
	for _, s := range c.Complete(":vim ") {
		fmt.Println(s)
	}

	// :arclm -init=true -initfn=test.inp test.otp
	c, _ = Compile(":arclm [init:$BOOL] [initfn:%g] %g",
		map[string][]string{
			"BOOL": []string{"true", "false"},
		})
	for _, s := range c.Complete(":arclm --initfn=c c") {
		fmt.Println( s)
	}
}
