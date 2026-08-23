//go:build linux && cgo

package webapi

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

typedef struct pam_handle pam_handle_t;
struct pam_message { int msg_style; const char *msg; };
struct pam_response { char *resp; int resp_retcode; };
struct pam_conv { int (*conv)(int, const struct pam_message **, struct pam_response **, void *); void *appdata_ptr; };

typedef int (*pam_start_fn)(const char *, const char *, const struct pam_conv *, pam_handle_t **);
typedef int (*pam_authenticate_fn)(pam_handle_t *, int);
typedef int (*pam_acct_mgmt_fn)(pam_handle_t *, int);
typedef int (*pam_end_fn)(pam_handle_t *, int);

static int dockflow_conv(int n, const struct pam_message **msg, struct pam_response **out, void *data) {
    struct pam_response *responses = calloc((size_t)n, sizeof(struct pam_response));
    if (!responses) return 5;
    for (int i = 0; i < n; i++) {
        if (msg[i]->msg_style == 1) responses[i].resp = strdup((const char *)data);
        else if (msg[i]->msg_style == 2) responses[i].resp = strdup("");
    }
    *out = responses;
    return 0;
}

static int dockflow_pam_auth(const char *user, const char *password) {
    void *lib = dlopen("libpam.so.0", RTLD_NOW);
    if (!lib) return -1;
    pam_start_fn start = (pam_start_fn)dlsym(lib, "pam_start");
    pam_authenticate_fn auth = (pam_authenticate_fn)dlsym(lib, "pam_authenticate");
    pam_acct_mgmt_fn account = (pam_acct_mgmt_fn)dlsym(lib, "pam_acct_mgmt");
    pam_end_fn end = (pam_end_fn)dlsym(lib, "pam_end");
    if (!start || !auth || !account || !end) { dlclose(lib); return -1; }
    struct pam_conv conv = { dockflow_conv, (void *)password };
    pam_handle_t *handle = NULL;
	int status = start("dockflow", user, &conv, &handle);
    if (status == 0) status = auth(handle, 0);
    if (status == 0) status = account(handle, 0);
    if (handle) end(handle, status);
    dlclose(lib);
    return status;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func authenticateSystemUser(username, password string) error {
	user := C.CString(username)
	defer C.free(unsafe.Pointer(user))
	pass := C.CString(password)
	defer C.free(unsafe.Pointer(pass))
	if C.dockflow_pam_auth(user, pass) != 0 {
		return errors.New("PAM authentication failed")
	}
	return nil
}
