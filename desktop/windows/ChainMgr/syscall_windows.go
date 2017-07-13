package main

import "syscall"

type tokenGroups struct {
	GroupCount uint32 // must be 2
	Groups     [2]syscall.SIDAndAttributes
}

//sys	AdjustTokenPrivileges(t syscall.Token, disableAllPrivs bool, newState unsafe.Pointer, newStateLen uint32, oldState unsafe.Pointer, oldStateLen uint32) (err error) = advapi32.AdjustTokenPrivileges
//sys	AdjustTokenGroups(t syscall.Token, resetToDefault bool, newState *tokenGroups, newStateLen uint32, oldState unsafe.Pointer, oldStateLen uint32) (err error) = advapi32.AdjustTokenGroups

func dropAdminPrivs() error {
	p, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}
	var t syscall.Token
	err = syscall.OpenProcessToken(p, syscall.TOKEN_ALL_ACCESS, &t)
	if err != nil {
		return err
	}

	err = AdjustTokenPrivileges(t, true, nil, 0, nil, 0)
	if err != nil {
		return err
	}

	tg := tokenGroups{GroupCount: 2}
	tg.Groups[0].Sid, err = syscall.StringToSid("S-1-5-32-544") // DOMAIN_ALIAS_RID_ADMINS = 0x0220
	if err != nil {
		return err
	}
	tg.Groups[1].Sid, err = syscall.StringToSid("S-1-5-32-547") // DOMAIN_ALIAS_RID_POWER_USERS = 0x0223
	if err != nil {
		return err
	}
	err = AdjustTokenGroups(t, false, &tg, 1, nil, 0)
	if err != nil {
		return err
	}
	return nil
}
