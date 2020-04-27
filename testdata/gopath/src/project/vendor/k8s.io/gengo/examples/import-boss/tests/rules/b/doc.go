// b only public and private packages. The latter it shouldn't.
package b

import (
	_ "k8s.io/gengo/examples/import-boss/tests/rules/c"
)
