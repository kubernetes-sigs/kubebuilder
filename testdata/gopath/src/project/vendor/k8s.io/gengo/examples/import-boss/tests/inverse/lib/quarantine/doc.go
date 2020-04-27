// quarantine is inside the library, but should not import the private package. But it does!
package quarantine

import (
	_ "k8s.io/gengo/examples/import-boss/tests/inverse/lib/private"
)
