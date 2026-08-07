package gui

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/permission"
)

// fakePermissionService records the permission decisions made through the
// dialog callbacks; the embedded interface satisfies the remaining methods
// (Subscribe, Request, AutoApproveSession) that the dialog flow never calls.
type fakePermissionService struct {
	permission.Service
	granted   []permission.PermissionRequest
	persisted []permission.PermissionRequest
	denied    []permission.PermissionRequest
}

func (f *fakePermissionService) Grant(p permission.PermissionRequest) {
	f.granted = append(f.granted, p)
}

func (f *fakePermissionService) GrantPersistant(p permission.PermissionRequest) {
	f.persisted = append(f.persisted, p)
}

func (f *fakePermissionService) Deny(p permission.PermissionRequest) {
	f.denied = append(f.denied, p)
}

func newPermissionWindow(t *testing.T, fake *fakePermissionService) (*MainWindow, *dialog.CustomDialog) {
	t.Helper()
	a := test.NewApp()
	win := a.NewWindow("permission-test")
	g := &MainWindow{core: &app.App{Permissions: fake}, win: win}
	dlg := dialog.NewCustom("t", "dismiss", container.NewVBox(), win)
	return g, dlg
}

func TestGrantPermissionApprovesOnce(t *testing.T) {
	fake := &fakePermissionService{}
	g, dlg := newPermissionWindow(t, fake)
	req := permission.PermissionRequest{ID: "p1", ToolName: "bash", Action: "ls"}

	g.grantPermission(dlg, req)

	if len(fake.granted) != 1 || fake.granted[0] != req {
		t.Fatalf("expected exactly one grant of the request, got %+v", fake.granted)
	}
	if len(fake.persisted) != 0 || len(fake.denied) != 0 {
		t.Fatal("grant must not persist or deny")
	}
}

func TestGrantPermissionForSessionPersists(t *testing.T) {
	fake := &fakePermissionService{}
	g, dlg := newPermissionWindow(t, fake)
	req := permission.PermissionRequest{ID: "p2", ToolName: "edit", Action: "write file"}

	g.grantPermissionForSession(dlg, req)

	if len(fake.persisted) != 1 || fake.persisted[0] != req {
		t.Fatalf("expected exactly one persisted grant, got %+v", fake.persisted)
	}
	if len(fake.granted) != 0 || len(fake.denied) != 0 {
		t.Fatal("session grant must not grant once or deny")
	}
}

func TestDenyPermissionRejects(t *testing.T) {
	fake := &fakePermissionService{}
	g, dlg := newPermissionWindow(t, fake)
	req := permission.PermissionRequest{ID: "p3", ToolName: "bash", Action: "rm -rf"}

	g.denyPermission(dlg, req)

	if len(fake.denied) != 1 || fake.denied[0] != req {
		t.Fatalf("expected exactly one denial, got %+v", fake.denied)
	}
	if len(fake.granted) != 0 || len(fake.persisted) != 0 {
		t.Fatal("deny must not grant or persist")
	}
}
