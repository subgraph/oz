package native

// #include <linux/audit.h>
import "C"

// AuditArch contains the architecture value for this architecture
const AuditArch = C.AUDIT_ARCH_X86_64
