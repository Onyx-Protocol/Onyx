package script

// Extension is a placeholder type until we define
// actual script extensions.
type Extension interface{}

const (
	// Evaluate P2SH subscripts (softfork safe, BIP16).
	// Called SCRIPT_VERIFY_P2SH in Bitcoin Core.
	FlagP2SH = 1 << iota

	// Passing a non-strict-DER signature or one with undefined
	// hashtype to a checksig operation causes script failure.
	// Evaluating a pubkey that is not (0x04 + 64 bytes) or
	// (0x02 or 0x03 + 32 bytes) by checksig causes script failure.
	// (softfork safe, but not used or intended as a consensus rule).
	// Called SCRIPT_VERIFY_STRICTENC in Bitcoin Core.
	FlagStrictEnc

	// Passing a non-strict-DER signature to a checksig operation
	// causes script failure (softfork safe, BIP62 rule 1)
	// Called SCRIPT_VERIFY_DERSIG in Bitcoin Core.
	FlagDERSig

	// Passing a non-strict-DER signature or one with S > order/2
	// to a checksig operation causes script failure
	// (softfork safe, BIP62 rule 5).
	// Called SCRIPT_VERIFY_LOW_S in Bitcoin Core.
	FlagLowS

	// Verify dummy stack item consumed by CHECKMULTISIG is of
	// zero-length (softfork safe, BIP62 rule 7).
	// Called SCRIPT_VERIFY_NULLDUMMY in Bitcoin Core.
	FlagNullDummy

	// Using a non-push operator in the scriptSig causes script failure
	// (softfork safe, BIP62 rule 2).
	// Called SCRIPT_VERIFY_SIGPUSHONLY in Bitcoin Core.
	FlagSigPushOnly

	// Require minimal encodings for all push operations
	// (OP_0... OP_16, OP_1NEGATE where possible, direct
	// pushes up to 75 bytes, OP_PUSHDATA up to 255 bytes,
	// OP_PUSHDATA2 for anything larger). Evaluating
	// any other push causes the script to fail (BIP62 rule 3).
	// In addition, whenever a stack element is interpreted
	// as a number, it must be of minimal length (BIP62 rule 4).
	// (softfork safe)
	// Called SCRIPT_VERIFY_MINIMALDATA in Bitcoin Core.
	FlagMinimalData

	// Discourage use of NOPs reserved for upgrades (NOP1-10)
	//
	// Provided so that nodes can avoid accepting or mining
	// transactions containing executed NOP's whose meaning may change
	// after a soft-fork, thus rendering the script invalid;
	// with this flag set executing discouraged NOPs fails the script.
	// This verification flag will never be a mandatory flag
	// applied to scripts in a block.
	//
	// NOPs that are not executed, e.g. within an unexecuted
	// IF ENDIF block, are *not* rejected.
	// Called SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS in Bitcoin Core.
	FlagDiscourageUpgradableNOPs

	// Require that only a single stack element remains after evaluation.
	// This changes the success criterion from
	// "At least one stack element must remain, and when interpreted
	//  as a boolean, it must be true" to
	// "Exactly one stack element must remain, and when interpreted as a
	// boolean, it must be true".
	// (softfork safe, BIP62 rule 6)
	// Note: CLEANSTACK should never be used without P2SH.
	// Called SCRIPT_VERIFY_CLEANSTACK in Bitcoin Core.
	FlagCleanStack

	// Verify CHECKLOCKTIMEVERIFY
	//
	// See BIP65 for details.
	// Called SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY in Bitcoin Core.
	FlagCHECKLOCKTIMEVERIFY

	// FlagNone is the zero value, meaning all optional
	// features are turned off.
	// Called SCRIPT_VERIFY_NONE in Bitcoin Core.
	FlagNone = 0
)

// DefaultFlags is a set of flags with all features turned on.
const DefaultFlags = FlagP2SH |
	FlagStrictEnc |
	FlagDERSig |
	FlagLowS |
	FlagNullDummy |
	FlagSigPushOnly |
	FlagMinimalData |
	FlagDiscourageUpgradableNOPs |
	FlagCleanStack |
	FlagCHECKLOCKTIMEVERIFY

// Params holds various configuration parameters for the script interpreter.
type Params struct {
	// Script flags
	Flags uint64

	// Extensions enable additional scripting features and opcodes.
	// Default extensions include P2SH (BIP16) and CLTV (BIP65).
	Extensions []Extension

	// Maximum allowed size of data pushed on stack.
	MaxPushdataSize int

	// Maximum allowed number of operations executed.
	// Multisig opcode counts as multiple operations.
	MaxOpCount int

	// Maximum allowed depth of the stack (both main stack and altstack).
	MaxStackSize int

	// Maximum allowed script size in bytes.
	MaxScriptSize int

	// Maximum bytesize of ScriptNumber for arithmetic operations.
	IntegerMaxSize int

	// Maximum bytesize of ScriptNumber for CLTV (BIP65) locktime checks.
	LockTimeMaxSize int

	// Helps with debugging: full stack trace is visible when script evaluation fails.
	PanicOnFailure bool
}
