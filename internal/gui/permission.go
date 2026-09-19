package gui

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/permission"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

func (g *MainWindow) showPermission(ev pubsub.Event[permission.PermissionRequest]) {
	tr := config.Tr()
	req := ev.Payload
	description := req.Description
	if description == "" {
		description = "(no description)"
	}

	allow := widget.NewButton(tr.PermAllow, nil)
	always := widget.NewButton(tr.PermAllowForSession, nil)
	deny := widget.NewButton(tr.PermDeny, nil)
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("%s: %s", tr.PermissionToolKey, req.ToolName)),
		widget.NewLabel(fmt.Sprintf("%s: %s", tr.PermissionCommandKey, req.Action)),
		widget.NewLabel(description),
		container.NewHBox(allow, always, deny),
	)
	dlg := dialog.NewCustom(tr.PermissionTitle, tr.PermDeny, content, g.win)
	// The dialog's dismiss button ("deny" text) only closes the window and
	// never answers the pending permission request, which would leave the tool
	// blocked forever. Treat any close without an explicit decision as a deny.
	handled := false
	allow.OnTapped = func() {
		handled = true
		g.grantPermission(dlg, req)
	}
	always.OnTapped = func() {
		handled = true
		g.grantPermissionForSession(dlg, req)
	}
	deny.OnTapped = func() {
		handled = true
		g.denyPermission(dlg, req)
	}
	dlg.SetOnClosed(func() {
		if !handled {
			g.core.Permissions.Deny(req)
		}
	})
	dlg.Show()
}

// grantPermission approves the request for this invocation only.
func (g *MainWindow) grantPermission(dlg *dialog.CustomDialog, req permission.PermissionRequest) {
	g.core.Permissions.Grant(req)
	dlg.Hide()
}

// grantPermissionForSession approves the request and remembers the choice for
// the current session.
func (g *MainWindow) grantPermissionForSession(dlg *dialog.CustomDialog, req permission.PermissionRequest) {
	g.core.Permissions.GrantPersistant(req)
	dlg.Hide()
}

// denyPermission rejects the request.
func (g *MainWindow) denyPermission(dlg *dialog.CustomDialog, req permission.PermissionRequest) {
	g.core.Permissions.Deny(req)
	dlg.Hide()
}
