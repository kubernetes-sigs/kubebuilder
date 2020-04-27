// c imports non-prod code. It shouldn't.
package d

import (
	_ "k8s.io/gengo/examples/import-boss/tests/inverse/lib/nonprod"
)
