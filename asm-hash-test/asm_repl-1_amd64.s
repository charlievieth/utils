#include "go_asm.h"
#include "funcdata.h"
#include "textflag.h"

TEXT ·aeshash(SB),NOSPLIT,$0-32
	JMP runtime·aeshash(SB)

TEXT ·aeshashstr(SB),NOSPLIT,$0-24
	JMP runtime·aeshashstr(SB)

TEXT ·aeshash32(SB),NOSPLIT,$0-24
	JMP runtime·aeshash32(SB)

TEXT ·aeshash64(SB),NOSPLIT,$0-24
	JMP runtime·aeshash64(SB)

