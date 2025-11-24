package featureflags

import "testing"

func TestComputeEffectivePrecedence(t *testing.T) {
    resetRegistry()
    RegisterFlags(Flag{Key: "f", Default: false})
    keys := []string{"f"}

    // no overrides → default
    out := computeEffective(keys, nil)
    if out["f"] != false {
        t.Fatalf("expected default false, got %v", out["f"])
    }

    // global true
    out = computeEffective(keys, []Override{{FlagKey: "f", PrincipalType: PrincipalGlobal, Value: true}})
    if !out["f"] { t.Fatalf("global true should enable") }

    // tenant overrides global
    out = computeEffective(keys, []Override{{FlagKey: "f", PrincipalType: PrincipalGlobal, Value: true}, {FlagKey: "f", PrincipalType: PrincipalTenant, Value: false}})
    if out["f"] { t.Fatalf("tenant false should override global true") }

    // role any=true; any true wins over false
    out = computeEffective(keys, []Override{{FlagKey: "f", PrincipalType: PrincipalRole, Value: false}, {FlagKey: "f", PrincipalType: PrincipalRole, Value: true}})
    if !out["f"] { t.Fatalf("any=true for roles should enable when any true") }

    // user overrides role/tenant/global
    out = computeEffective(keys, []Override{{FlagKey: "f", PrincipalType: PrincipalGlobal, Value: false}, {FlagKey: "f", PrincipalType: PrincipalTenant, Value: true}, {FlagKey: "f", PrincipalType: PrincipalRole, Value: false}, {FlagKey: "f", PrincipalType: PrincipalUser, Value: false}})
    if out["f"] { t.Fatalf("user false should override others") }
}
