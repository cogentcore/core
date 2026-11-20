// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wintoast

import (
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procRoActivateInstance     = combase.NewProc("RoActivateInstance")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")

	iidIToastNotificationManagerStatics   = guid("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
	iidIToastNotificationFactory          = guid("{04124B20-82C6-4229-B109-FD9ED4662B53}")
	iidIScheduledToastNotificationFactory = guid("{E7BED191-0BB9-4189-8394-31761B476FD7}")
	iidIScheduledToastNotification        = guid("{79F577F8-0DE7-48CD-9740-9B370490C838}")
	iidIToastNotification2                = guid("{9DFB9FD1-143A-490E-90BF-B9FBA7132DE7}")
	iidIXmlDocument                       = guid("{F7F3A506-1E87-42D6-BCFB-B8C809FA5494}")
	iidIXmlDocumentIO                     = guid("{6CD0E74E-EE65-4489-9EBF-CA43E87BA637}")
)

// The runtime classes.
const (
	classToastNotificationManager   = "Windows.UI.Notifications.ToastNotificationManager"
	classToastNotification          = "Windows.UI.Notifications.ToastNotification"
	classScheduledToastNotification = "Windows.UI.Notifications.ScheduledToastNotification"
	classXmlDocument                = "Windows.Data.Xml.Dom.XmlDocument"
)

// guid parses a GUID string, which must be valid.
func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// COM vtable method indexes, in the order declared in the WinRT headers
// (windows.ui.notifications.h, windows.data.xml.dom.h), after the methods
// of IUnknown (0-2) and IInspectable (3-5).
const (
	vtQueryInterface = 0
	vtRelease        = 2

	// IToastNotificationManagerStatics
	vtCreateToastNotifierWithID = 7

	// IXmlDocumentIO
	vtLoadXML = 6

	// IToastNotificationFactory, IScheduledToastNotificationFactory
	vtCreate = 6

	// IToastNotifier
	vtShow                           = 6
	vtGetSetting                     = 8
	vtAddToSchedule                  = 9
	vtRemoveFromSchedule             = 10
	vtGetScheduledToastNotifications = 11

	// IScheduledToastNotification
	vtPutID = 10

	// IToastNotification2
	vtPutTag = 6

	// IVectorView
	vtGetAt   = 6
	vtGetSize = 7
)

// comObject is a pointer to a COM object, whose first field
// is a pointer to its vtable.
type comObject = *struct {
	vtbl *[32]uintptr
}

// comCall calls the vtable method at index i with the object and args,
// returning an error for a failed HRESULT.
func comCall(o comObject, i int, args ...uintptr) error {
	r, _, _ := syscall.SyscallN(o.vtbl[i], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	if int32(r) < 0 {
		return fmt.Errorf("wintoast: COM call failed: HRESULT 0x%08X", uint32(r))
	}
	return nil
}

// comRelease releases the object if it is not nil.
func comRelease(o comObject) {
	if o != nil {
		comCall(o, vtRelease)
	}
}

// queryInterface returns the interface of the object with the given IID.
func queryInterface(o comObject, iid *windows.GUID) (comObject, error) {
	var q comObject
	err := comCall(o, vtQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&q)))
	return q, err
}

// hstring is a WinRT string handle.
type hstring uintptr

// newHString returns a new WinRT string, which must be deleted.
func newHString(s string) (hstring, error) {
	u := utf16.Encode([]rune(s))
	var p *uint16
	if len(u) > 0 {
		p = &u[0]
	}
	var h hstring
	r, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(p)), uintptr(len(u)), uintptr(unsafe.Pointer(&h)))
	if int32(r) < 0 {
		return 0, fmt.Errorf("wintoast: WindowsCreateString failed: HRESULT 0x%08X", uint32(r))
	}
	return h, nil
}

func (h hstring) delete() {
	if h != 0 {
		procWindowsDeleteString.Call(uintptr(h))
	}
}

// activationFactory returns the activation factory of the given runtime
// class, with the given interface.
func activationFactory(class string, iid *windows.GUID) (comObject, error) {
	hc, err := newHString(class)
	if err != nil {
		return nil, err
	}
	defer hc.delete()
	var f comObject
	r, _, _ := procRoGetActivationFactory.Call(uintptr(hc), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&f)))
	if int32(r) < 0 {
		return nil, fmt.Errorf("wintoast: RoGetActivationFactory(%s) failed: HRESULT 0x%08X", class, uint32(r))
	}
	return f, nil
}

// Notifier shows and schedules toasts for an app, by its AppUserModelID.
// All of the COM calls are made on a single locked OS thread that has
// initialized COM, through funcs sent on the calls channel.
type Notifier struct {
	calls chan func()

	notifier  comObject // IToastNotifier
	toasts    comObject // IToastNotificationFactory
	scheduled comObject // IScheduledToastNotificationFactory
}

// New returns a new [Notifier] for the app with the given AppUserModelID,
// which is registered for the current user with the given display name,
// as needed for the toasts of an app that is not packaged.
func New(aumid, displayName string) (*Notifier, error) {
	if err := register(aumid, displayName); err != nil {
		return nil, err
	}
	nt := &Notifier{calls: make(chan func())}
	errc := make(chan error)
	go nt.run(aumid, errc)
	if err := <-errc; err != nil {
		return nil, err
	}
	return nt, nil
}

// register registers the AppUserModelID for the current user, with the
// display name of the toasts.
func register(aumid, displayName string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+aumid, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("wintoast: registering the AppUserModelID: %w", err)
	}
	defer k.Close()
	return k.SetStringValue("DisplayName", displayName)
}

// run initializes COM and the notifier, sending the result to errc,
// and then serves calls.
func (nt *Notifier) run(aumid string, errc chan error) {
	runtime.LockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil {
		errc <- err
		return
	}
	if err := nt.init(aumid); err != nil {
		windows.CoUninitialize()
		errc <- err
		return
	}
	errc <- nil
	for f := range nt.calls {
		f()
	}
}

// init makes the toast notifier and the factories of the toasts.
func (nt *Notifier) init(aumid string) error {
	statics, err := activationFactory(classToastNotificationManager, &iidIToastNotificationManagerStatics)
	if err != nil {
		return err
	}
	defer comRelease(statics)
	ha, err := newHString(aumid)
	if err != nil {
		return err
	}
	defer ha.delete()
	if err := comCall(statics, vtCreateToastNotifierWithID, uintptr(ha), uintptr(unsafe.Pointer(&nt.notifier))); err != nil {
		return err
	}
	if nt.toasts, err = activationFactory(classToastNotification, &iidIToastNotificationFactory); err != nil {
		return err
	}
	nt.scheduled, err = activationFactory(classScheduledToastNotification, &iidIScheduledToastNotificationFactory)
	return err
}

// call runs the given function on the COM thread, returning its error.
func (nt *Notifier) call(f func() error) error {
	errc := make(chan error)
	nt.calls <- func() { errc <- f() }
	return <-errc
}

// Enabled returns whether the toasts of the app are enabled, which is not
// the case when the user turned off its notifications.
func (nt *Notifier) Enabled() bool {
	var setting int32
	err := nt.call(func() error {
		return comCall(nt.notifier, vtGetSetting, uintptr(unsafe.Pointer(&setting)))
	})
	return err == nil && setting == 0 // NotificationSetting_Enabled
}

// shortID returns a short ID for the toast with the given ID, as the ID
// of a scheduled toast has at most 16 characters (and a tag 64).
func shortID(id string) string {
	h := fnv.New64a()
	h.Write([]byte(id))
	return fmt.Sprintf("%016x", h.Sum64())
}

// xmlContent returns the XML document of the content of the given toast,
// whose launch argument is its ID.
func xmlContent(t *Toast) (comObject, error) {
	hc, err := newHString(classXmlDocument)
	if err != nil {
		return nil, err
	}
	defer hc.delete()
	var doc comObject
	r, _, _ := procRoActivateInstance.Call(uintptr(hc), uintptr(unsafe.Pointer(&doc)))
	if int32(r) < 0 {
		return nil, fmt.Errorf("wintoast: RoActivateInstance(%s) failed: HRESULT 0x%08X", classXmlDocument, uint32(r))
	}
	defer comRelease(doc)
	io, err := queryInterface(doc, &iidIXmlDocumentIO)
	if err != nil {
		return nil, err
	}
	defer comRelease(io)
	x := `<toast launch="` + html.EscapeString(t.ID) + `"><visual><binding template="ToastGeneric"><text>` +
		html.EscapeString(t.Title) + `</text><text>` + html.EscapeString(t.Body) + `</text></binding></visual></toast>`
	hx, err := newHString(x)
	if err != nil {
		return nil, err
	}
	defer hx.delete()
	if err := comCall(io, vtLoadXML, uintptr(hx)); err != nil {
		return nil, err
	}
	return queryInterface(doc, &iidIXmlDocument)
}

// Show shows the given toast now (its time is not used).
func (nt *Notifier) Show(t *Toast) error {
	return nt.call(func() error {
		content, err := xmlContent(t)
		if err != nil {
			return err
		}
		defer comRelease(content)
		var toast comObject
		if err := comCall(nt.toasts, vtCreate, uintptr(unsafe.Pointer(content)), uintptr(unsafe.Pointer(&toast))); err != nil {
			return err
		}
		defer comRelease(toast)
		// the tag makes a toast with the same ID replace it
		if t2, err := queryInterface(toast, &iidIToastNotification2); err == nil {
			if ht, err := newHString(shortID(t.ID)); err == nil {
				comCall(t2, vtPutTag, uintptr(ht))
				ht.delete()
			}
			comRelease(t2)
		}
		return comCall(nt.notifier, vtShow, uintptr(unsafe.Pointer(toast)))
	})
}

// Sync schedules the given toasts at their times, removing all of the
// toasts scheduled before for the app, which Windows shows at their times
// also when the app is not running.
func (nt *Notifier) Sync(ts []*Toast) error {
	return nt.call(func() error {
		var errs []error
		if err := nt.removeScheduled(); err != nil {
			errs = append(errs, err)
		}
		for _, t := range ts {
			if err := nt.schedule(t); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	})
}

// removeScheduled removes all of the scheduled toasts of the app.
func (nt *Notifier) removeScheduled() error {
	var view comObject
	if err := comCall(nt.notifier, vtGetScheduledToastNotifications, uintptr(unsafe.Pointer(&view))); err != nil {
		return err
	}
	defer comRelease(view)
	var n uint32
	if err := comCall(view, vtGetSize, uintptr(unsafe.Pointer(&n))); err != nil {
		return err
	}
	var errs []error
	for i := range n {
		var st comObject
		if err := comCall(view, vtGetAt, uintptr(i), uintptr(unsafe.Pointer(&st))); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := comCall(nt.notifier, vtRemoveFromSchedule, uintptr(unsafe.Pointer(st))); err != nil {
			errs = append(errs, err)
		}
		comRelease(st)
	}
	return errors.Join(errs...)
}

// schedule schedules the given toast at its time.
func (nt *Notifier) schedule(t *Toast) error {
	content, err := xmlContent(t)
	if err != nil {
		return err
	}
	defer comRelease(content)
	// a DateTime is the number of 100ns intervals since 1601, passed by
	// value, in two arguments on 32-bit systems
	dt := t.Time.UnixNano()/100 + 116444736000000000
	args := []uintptr{uintptr(unsafe.Pointer(content))}
	if unsafe.Sizeof(uintptr(0)) == 4 {
		args = append(args, uintptr(uint32(dt)), uintptr(uint32(dt>>32)))
	} else {
		args = append(args, uintptr(dt))
	}
	var st comObject
	args = append(args, uintptr(unsafe.Pointer(&st)))
	if err := comCall(nt.scheduled, vtCreate, args...); err != nil {
		return err
	}
	defer comRelease(st)
	if sti, err := queryInterface(st, &iidIScheduledToastNotification); err == nil {
		if hi, err := newHString(shortID(t.ID)); err == nil {
			comCall(sti, vtPutID, uintptr(hi))
			hi.delete()
		}
		comRelease(sti)
	}
	return comCall(nt.notifier, vtAddToSchedule, uintptr(unsafe.Pointer(st)))
}

// NumScheduled returns the number of the scheduled toasts of the app.
func (nt *Notifier) NumScheduled() (int, error) {
	var n uint32
	err := nt.call(func() error {
		var view comObject
		if err := comCall(nt.notifier, vtGetScheduledToastNotifications, uintptr(unsafe.Pointer(&view))); err != nil {
			return err
		}
		defer comRelease(view)
		return comCall(view, vtGetSize, uintptr(unsafe.Pointer(&n)))
	})
	return int(n), err
}
