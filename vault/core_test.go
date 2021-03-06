package vault

import (
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/vault/audit"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical"
)

var (
	// invalidKey is used to test Unseal
	invalidKey = []byte("abcdefghijklmnopqrstuvwxyz")[:17]
)

func TestCore_Init(t *testing.T) {
	inm := physical.NewInmem()
	conf := &CoreConfig{Physical: inm}
	c, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	init, err := c.Initialized()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if init {
		t.Fatalf("should not be init")
	}

	// Check the seal configuration
	outConf, err := c.SealConfig()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if outConf != nil {
		t.Fatalf("bad: %v", outConf)
	}

	sealConf := &SealConfig{
		SecretShares:    1,
		SecretThreshold: 1,
	}
	res, err := c.Initialize(sealConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(res.SecretShares) != 1 {
		t.Fatalf("Bad: %v", res)
	}
	if res.RootToken == "" {
		t.Fatalf("Bad: %v", res)
	}

	_, err = c.Initialize(sealConf)
	if err != ErrAlreadyInit {
		t.Fatalf("err: %v", err)
	}

	init, err = c.Initialized()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !init {
		t.Fatalf("should be init")
	}

	// Check the seal configuration
	outConf, err = c.SealConfig()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(outConf, sealConf) {
		t.Fatalf("bad: %v expect: %v", outConf, sealConf)
	}

	// New Core, same backend
	c2, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	_, err = c2.Initialize(sealConf)
	if err != ErrAlreadyInit {
		t.Fatalf("err: %v", err)
	}

	init, err = c2.Initialized()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !init {
		t.Fatalf("should be init")
	}

	// Check the seal configuration
	outConf, err = c2.SealConfig()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(outConf, sealConf) {
		t.Fatalf("bad: %v expect: %v", outConf, sealConf)
	}
}

func TestCore_Init_MultiShare(t *testing.T) {
	c := TestCore(t)
	sealConf := &SealConfig{
		SecretShares:    5,
		SecretThreshold: 3,
	}
	res, err := c.Initialize(sealConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(res.SecretShares) != 5 {
		t.Fatalf("Bad: %v", res)
	}
	if res.RootToken == "" {
		t.Fatalf("Bad: %v", res)
	}

	// Check the seal configuration
	outConf, err := c.SealConfig()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(outConf, sealConf) {
		t.Fatalf("bad: %v expect: %v", outConf, sealConf)
	}
}

func TestCore_Unseal_MultiShare(t *testing.T) {
	c := TestCore(t)

	_, err := c.Unseal(invalidKey)
	if err != ErrNotInit {
		t.Fatalf("err: %v", err)
	}

	sealConf := &SealConfig{
		SecretShares:    5,
		SecretThreshold: 3,
	}
	res, err := c.Initialize(sealConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sealed, err := c.Sealed()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !sealed {
		t.Fatalf("should be sealed")
	}

	if prog := c.SecretProgress(); prog != 0 {
		t.Fatalf("bad progress: %d", prog)
	}

	for i := 0; i < 5; i++ {
		unseal, err := c.Unseal(res.SecretShares[i])
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		// Ignore redundant
		_, err = c.Unseal(res.SecretShares[i])
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if i >= 2 {
			if !unseal {
				t.Fatalf("should be unsealed")
			}
			if prog := c.SecretProgress(); prog != 0 {
				t.Fatalf("bad progress: %d", prog)
			}
		} else {
			if unseal {
				t.Fatalf("should not be unsealed")
			}
			if prog := c.SecretProgress(); prog != i+1 {
				t.Fatalf("bad progress: %d", prog)
			}
		}
	}

	sealed, err = c.Sealed()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sealed {
		t.Fatalf("should not be sealed")
	}

	err = c.Seal(res.RootToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ignore redundant
	err = c.Seal(res.RootToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sealed, err = c.Sealed()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !sealed {
		t.Fatalf("should be sealed")
	}
}

func TestCore_Unseal_Single(t *testing.T) {
	c := TestCore(t)

	_, err := c.Unseal(invalidKey)
	if err != ErrNotInit {
		t.Fatalf("err: %v", err)
	}

	sealConf := &SealConfig{
		SecretShares:    1,
		SecretThreshold: 1,
	}
	res, err := c.Initialize(sealConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sealed, err := c.Sealed()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !sealed {
		t.Fatalf("should be sealed")
	}

	if prog := c.SecretProgress(); prog != 0 {
		t.Fatalf("bad progress: %d", prog)
	}

	unseal, err := c.Unseal(res.SecretShares[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !unseal {
		t.Fatalf("should be unsealed")
	}
	if prog := c.SecretProgress(); prog != 0 {
		t.Fatalf("bad progress: %d", prog)
	}

	sealed, err = c.Sealed()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sealed {
		t.Fatalf("should not be sealed")
	}
}

func TestCore_Route_Sealed(t *testing.T) {
	c := TestCore(t)
	sealConf := &SealConfig{
		SecretShares:    1,
		SecretThreshold: 1,
	}

	// Should not route anything
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "sys/mounts",
	}
	_, err := c.HandleRequest(req)
	if err != ErrSealed {
		t.Fatalf("err: %v", err)
	}

	res, err := c.Initialize(sealConf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	unseal, err := c.Unseal(res.SecretShares[0])
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !unseal {
		t.Fatalf("should be unsealed")
	}

	// Should not error after unseal
	req.ClientToken = res.RootToken
	_, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

// Attempt to unseal after doing a first seal
func TestCore_SealUnseal(t *testing.T) {
	c, key, root := TestCoreUnsealed(t)
	if err := c.Seal(root); err != nil {
		t.Fatalf("err: %v", err)
	}
	if unseal, err := c.Unseal(key); err != nil || !unseal {
		t.Fatalf("err: %v", err)
	}
}

// Attempt to seal bad token
func TestCore_Seal_BadToken(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)
	if err := c.Seal("foo"); err == nil {
		t.Fatalf("err: %v", err)
	}
	if sealed, err := c.Sealed(); err != nil || sealed {
		t.Fatalf("err: %v", err)
	}
}

// Ensure we get a LeaseID
func TestCore_HandleRequest_Lease(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	// Read the key
	req.Operation = logical.ReadOperation
	req.Data = nil
	resp, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil || resp.Secret == nil || resp.Data == nil {
		t.Fatalf("bad: %#v", resp)
	}
	if resp.Secret.Lease != time.Hour {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Secret.LeaseID == "" {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Data["foo"] != "bar" {
		t.Fatalf("bad: %#v", resp.Data)
	}
}

func TestCore_HandleRequest_Lease_MaxLength(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1000h",
		},
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	// Read the key
	req.Operation = logical.ReadOperation
	req.Data = nil
	resp, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil || resp.Secret == nil || resp.Data == nil {
		t.Fatalf("bad: %#v", resp)
	}
	if resp.Secret.Lease != maxLeaseDuration {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Secret.LeaseID == "" {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Data["foo"] != "bar" {
		t.Fatalf("bad: %#v", resp.Data)
	}
}

func TestCore_HandleRequest_Lease_DefaultLength(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "0h",
		},
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	// Read the key
	req.Operation = logical.ReadOperation
	req.Data = nil
	resp, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil || resp.Secret == nil || resp.Data == nil {
		t.Fatalf("bad: %#v", resp)
	}
	if resp.Secret.Lease != defaultLeaseDuration {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Secret.LeaseID == "" {
		t.Fatalf("bad: %#v", resp.Secret)
	}
	if resp.Data["foo"] != "bar" {
		t.Fatalf("bad: %#v", resp.Data)
	}
}

func TestCore_HandleRequest_MissingToken(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
	}
	resp, err := c.HandleRequest(req)
	if err != logical.ErrInvalidRequest {
		t.Fatalf("err: %v", err)
	}
	if resp.Data["error"] != "missing client token" {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestCore_HandleRequest_InvalidToken(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
		ClientToken: "foobarbaz",
	}
	resp, err := c.HandleRequest(req)
	if err != logical.ErrPermissionDenied {
		t.Fatalf("err: %v", err)
	}
	if resp.Data["error"] != "permission denied" {
		t.Fatalf("bad: %#v", resp)
	}
}

// Check that standard permissions work
func TestCore_HandleRequest_NoSlash(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	req := &logical.Request{
		Operation:   logical.HelpOperation,
		Path:        "secret",
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v, resp: %v", err, resp)
	}
	if _, ok := resp.Data["help"]; !ok {
		t.Fatalf("resp: %v", resp)
	}
}

// Test a root path is denied if non-root
func TestCore_HandleRequest_RootPath(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)
	testCoreMakeToken(t, c, root, "child", []string{"test"})

	req := &logical.Request{
		Operation:   logical.ReadOperation,
		Path:        "sys/policy", // root protected!
		ClientToken: "child",
	}
	resp, err := c.HandleRequest(req)
	if err != logical.ErrPermissionDenied {
		t.Fatalf("err: %v, resp: %v", err, resp)
	}
}

// Test a root path is allowed if non-root but with sudo
func TestCore_HandleRequest_RootPath_WithSudo(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	// Set the 'test' policy object to permit access to sys/policy
	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "sys/policy/test", // root protected!
		Data: map[string]interface{}{
			"rules": `path "sys/policy" { policy = "sudo" }`,
		},
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	// Child token (non-root) but with 'test' policy should have access
	testCoreMakeToken(t, c, root, "child", []string{"test"})
	req = &logical.Request{
		Operation:   logical.ReadOperation,
		Path:        "sys/policy", // root protected!
		ClientToken: "child",
	}
	resp, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil {
		t.Fatalf("bad: %#v", resp)
	}
}

// Check that standard permissions work
func TestCore_HandleRequest_PermissionDenied(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)
	testCoreMakeToken(t, c, root, "child", []string{"test"})

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
		ClientToken: "child",
	}
	resp, err := c.HandleRequest(req)
	if err != logical.ErrPermissionDenied {
		t.Fatalf("err: %v, resp: %v", err, resp)
	}
}

// Check that standard permissions work
func TestCore_HandleRequest_PermissionAllowed(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)
	testCoreMakeToken(t, c, root, "child", []string{"test"})

	// Set the 'test' policy object to permit access to secret/
	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "sys/policy/test",
		Data: map[string]interface{}{
			"rules": `path "secret/" { policy = "write" }`,
		},
		ClientToken: root,
	}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}

	// Write should work now
	req = &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
		ClientToken: "child",
	}
	resp, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp != nil {
		t.Fatalf("bad: %#v", resp)
	}
}

func TestCore_HandleRequest_NoConnection(t *testing.T) {
	noop := &NoopBackend{
		Response: &logical.Response{},
	}
	c, _, root := TestCoreUnsealed(t)
	c.logicalBackends["noop"] = func(map[string]string) (logical.Backend, error) {
		return noop, nil
	}

	// Enable the logical backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/mounts/foo")
	req.Data["type"] = "noop"
	req.Data["description"] = "foo"
	req.ClientToken = root
	_, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Attempt to request with connection data
	req = &logical.Request{
		Path:       "foo/login",
		Connection: &logical.Connection{},
	}
	req.ClientToken = root
	if _, err := c.HandleRequest(req); err != nil {
		t.Fatalf("err: %v", err)
	}
	if noop.Requests[0].Connection != nil {
		t.Fatalf("bad: %#v", noop.Requests)
	}
}

func TestCore_HandleRequest_NoClientToken(t *testing.T) {
	noop := &NoopBackend{
		Response: &logical.Response{},
	}
	c, _, root := TestCoreUnsealed(t)
	c.logicalBackends["noop"] = func(map[string]string) (logical.Backend, error) {
		return noop, nil
	}

	// Enable the logical backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/mounts/foo")
	req.Data["type"] = "noop"
	req.Data["description"] = "foo"
	req.ClientToken = root
	_, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Attempt to request with connection data
	req = &logical.Request{
		Path: "foo/login",
	}
	req.ClientToken = root
	if _, err := c.HandleRequest(req); err != nil {
		t.Fatalf("err: %v", err)
	}

	ct := noop.Requests[0].ClientToken
	if ct == "" || ct == root {
		t.Fatalf("bad: %#v", noop.Requests)
	}
}

func TestCore_HandleRequest_ConnOnLogin(t *testing.T) {
	noop := &NoopBackend{
		Login:    []string{"login"},
		Response: &logical.Response{},
	}
	c, _, root := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(map[string]string) (logical.Backend, error) {
		return noop, nil
	}

	// Enable the credential backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/auth/foo")
	req.Data["type"] = "noop"
	req.ClientToken = root
	_, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Attempt to request with connection data
	req = &logical.Request{
		Path:       "auth/foo/login",
		Connection: &logical.Connection{},
	}
	if _, err := c.HandleRequest(req); err != nil {
		t.Fatalf("err: %v", err)
	}
	if noop.Requests[0].Connection == nil {
		t.Fatalf("bad: %#v", noop.Requests)
	}
}

// Ensure we get a client token
func TestCore_HandleLogin_Token(t *testing.T) {
	noop := &NoopBackend{
		Login: []string{"login"},
		Response: &logical.Response{
			Auth: &logical.Auth{
				Policies: []string{"foo", "bar"},
				Metadata: map[string]string{
					"user": "armon",
				},
				DisplayName: "armon",
			},
		},
	}
	c, _, root := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(map[string]string) (logical.Backend, error) {
		return noop, nil
	}

	// Enable the credential backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/auth/foo")
	req.Data["type"] = "noop"
	req.ClientToken = root
	_, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Attempt to login
	lreq := &logical.Request{
		Path: "auth/foo/login",
	}
	lresp, err := c.HandleRequest(lreq)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we got a client token back
	clientToken := lresp.Auth.ClientToken
	if clientToken == "" {
		t.Fatalf("bad: %#v", lresp)
	}

	// Check the policy and metadata
	te, err := c.tokenStore.Lookup(clientToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expect := &TokenEntry{
		ID:       clientToken,
		Parent:   "",
		Policies: []string{"foo", "bar"},
		Path:     "auth/foo/login",
		Meta: map[string]string{
			"user": "armon",
		},
		DisplayName: "foo-armon",
	}
	if !reflect.DeepEqual(te, expect) {
		t.Fatalf("Bad: %#v expect: %#v", te, expect)
	}

	// Check that we have a lease with default duration
	if lresp.Auth.Lease != defaultLeaseDuration {
		t.Fatalf("bad: %#v", lresp.Auth)
	}
}

func TestCore_HandleRequest_AuditTrail(t *testing.T) {
	// Create a noop audit backend
	noop := &NoopAudit{}
	c, _, root := TestCoreUnsealed(t)
	c.auditBackends["noop"] = func(map[string]string) (audit.Backend, error) {
		return noop, nil
	}

	// Enable the audit backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/audit/noop")
	req.Data["type"] = "noop"
	req.ClientToken = root
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Make a request
	req = &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/test",
		Data: map[string]interface{}{
			"foo":   "bar",
			"lease": "1h",
		},
		ClientToken: root,
	}
	req.ClientToken = root
	if _, err := c.HandleRequest(req); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the audit trail on request and response
	if len(noop.ReqAuth) != 1 {
		t.Fatalf("bad: %#v", noop)
	}
	auth := noop.ReqAuth[0]
	if auth.ClientToken != root {
		t.Fatalf("bad client token: %#v", auth)
	}
	if len(auth.Policies) != 1 || auth.Policies[0] != "root" {
		t.Fatalf("bad: %#v", auth)
	}
	if len(noop.Req) != 1 || !reflect.DeepEqual(noop.Req[0], req) {
		t.Fatalf("Bad: %#v", noop.Req[0])
	}

	if len(noop.RespAuth) != 2 {
		t.Fatalf("bad: %#v", noop)
	}
	if !reflect.DeepEqual(noop.RespAuth[1], auth) {
		t.Fatalf("bad: %#v", auth)
	}
	if len(noop.RespReq) != 2 || !reflect.DeepEqual(noop.RespReq[1], req) {
		t.Fatalf("Bad: %#v", noop.RespReq[1])
	}
	if len(noop.Resp) != 2 || !reflect.DeepEqual(noop.Resp[1], resp) {
		t.Fatalf("Bad: %#v", noop.Resp[1])
	}
}

// Ensure we get a client token
func TestCore_HandleLogin_AuditTrail(t *testing.T) {
	// Create a badass credential backend that always logs in as armon
	noop := &NoopAudit{}
	noopBack := &NoopBackend{
		Login: []string{"login"},
		Response: &logical.Response{
			Secret: &logical.Secret{
				LeaseOptions: logical.LeaseOptions{
					Lease: time.Hour,
				},
			},

			Auth: &logical.Auth{
				Policies: []string{"foo", "bar"},
				Metadata: map[string]string{
					"user": "armon",
				},
			},
		},
	}
	c, _, root := TestCoreUnsealed(t)
	c.credentialBackends["noop"] = func(map[string]string) (logical.Backend, error) {
		return noopBack, nil
	}
	c.auditBackends["noop"] = func(map[string]string) (audit.Backend, error) {
		return noop, nil
	}

	// Enable the credential backend
	req := logical.TestRequest(t, logical.WriteOperation, "sys/auth/foo")
	req.Data["type"] = "noop"
	req.ClientToken = root
	_, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Enable the audit backend
	req = logical.TestRequest(t, logical.WriteOperation, "sys/audit/noop")
	req.Data["type"] = "noop"
	req.ClientToken = root
	_, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Attempt to login
	lreq := &logical.Request{
		Path: "auth/foo/login",
	}
	lresp, err := c.HandleRequest(lreq)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we got a client token back
	clientToken := lresp.Auth.ClientToken
	if clientToken == "" {
		t.Fatalf("bad: %#v", lresp)
	}

	// Check the audit trail on request and response
	if len(noop.ReqAuth) != 1 {
		t.Fatalf("bad: %#v", noop)
	}
	if len(noop.Req) != 1 || !reflect.DeepEqual(noop.Req[0], lreq) {
		t.Fatalf("Bad: %#v %#v", noop.Req[0], lreq)
	}

	if len(noop.RespAuth) != 2 {
		t.Fatalf("bad: %#v", noop)
	}
	auth := noop.RespAuth[1]
	if auth.ClientToken != clientToken {
		t.Fatalf("bad client token: %#v", auth)
	}
	if len(auth.Policies) != 2 || auth.Policies[0] != "foo" || auth.Policies[1] != "bar" {
		t.Fatalf("bad: %#v", auth)
	}
	if len(noop.RespReq) != 2 || !reflect.DeepEqual(noop.RespReq[1], lreq) {
		t.Fatalf("Bad: %#v", noop.RespReq[1])
	}
	if len(noop.Resp) != 2 || !reflect.DeepEqual(noop.Resp[1], lresp) {
		t.Fatalf("Bad: %#v %#v", noop.Resp[1], lresp)
	}
}

// Check that we register a lease for new tokens
func TestCore_HandleRequest_CreateToken_Lease(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	// Create a new credential
	req := logical.TestRequest(t, logical.WriteOperation, "auth/token/create")
	req.ClientToken = root
	req.Data["policies"] = []string{"foo"}
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we got a new client token back
	clientToken := resp.Auth.ClientToken
	if clientToken == "" {
		t.Fatalf("bad: %#v", resp)
	}

	// Check the policy and metadata
	te, err := c.tokenStore.Lookup(clientToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expect := &TokenEntry{
		ID:          clientToken,
		Parent:      root,
		Policies:    []string{"foo"},
		Path:        "auth/token/create",
		DisplayName: "token",
	}
	if !reflect.DeepEqual(te, expect) {
		t.Fatalf("Bad: %#v expect: %#v", te, expect)
	}

	// Check that we have a lease with default duration
	if resp.Auth.Lease != defaultLeaseDuration {
		t.Fatalf("bad: %#v", resp.Auth)
	}
}

func TestCore_LimitedUseToken(t *testing.T) {
	c, _, root := TestCoreUnsealed(t)

	// Create a new credential
	req := logical.TestRequest(t, logical.WriteOperation, "auth/token/create")
	req.ClientToken = root
	req.Data["num_uses"] = "1"
	resp, err := c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Put a secret
	req = &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/foo",
		Data: map[string]interface{}{
			"foo": "bar",
		},
		ClientToken: resp.Auth.ClientToken,
	}
	_, err = c.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Second operation should fail
	_, err = c.HandleRequest(req)
	if err != logical.ErrPermissionDenied {
		t.Fatalf("err: %v", err)
	}
}

func TestCore_Standby(t *testing.T) {
	// Create the first core and initialize it
	inm := physical.NewInmemHA()
	core, err := NewCore(&CoreConfig{Physical: inm, AdvertiseAddr: "foo"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	key, root := TestCoreInit(t, core)
	if _, err := core.Unseal(TestKeyCopy(key)); err != nil {
		t.Fatalf("unseal err: %s", err)
	}

	// Verify unsealed
	sealed, err := core.Sealed()
	if err != nil {
		t.Fatalf("err checking seal status: %s", err)
	}
	if sealed {
		t.Fatal("should not be sealed")
	}

	// Wait for core to become active
	start := time.Now()
	var standby bool
	for time.Now().Sub(start) < time.Second {
		standby, err = core.Standby()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !standby {
			break
		}
	}
	if standby {
		t.Fatalf("should not be in standby mode")
	}

	// Put a secret
	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/foo",
		Data: map[string]interface{}{
			"foo": "bar",
		},
		ClientToken: root,
	}
	_, err = core.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the leader is local
	isLeader, advertise, err := core.Leader()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isLeader {
		t.Fatalf("should be leader")
	}
	if advertise != "foo" {
		t.Fatalf("Bad advertise: %v", advertise)
	}

	// Create a second core, attached to same in-memory store
	core2, err := NewCore(&CoreConfig{Physical: inm, AdvertiseAddr: "bar"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := core2.Unseal(TestKeyCopy(key)); err != nil {
		t.Fatalf("unseal err: %s", err)
	}

	// Verify unsealed
	sealed, err = core2.Sealed()
	if err != nil {
		t.Fatalf("err checking seal status: %s", err)
	}
	if sealed {
		t.Fatal("should not be sealed")
	}

	// Core2 should be in standby
	standby, err = core2.Standby()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !standby {
		t.Fatalf("should be standby")
	}

	// Request should fail in standby mode
	_, err = core2.HandleRequest(req)
	if err != ErrStandby {
		t.Fatalf("err: %v", err)
	}

	// Check the leader is not local
	isLeader, advertise, err = core2.Leader()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isLeader {
		t.Fatalf("should not be leader")
	}
	if advertise != "foo" {
		t.Fatalf("Bad advertise: %v", advertise)
	}

	// Seal the first core, should step down
	err = core.Seal(root)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Core should be in standby
	standby, err = core.Standby()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !standby {
		t.Fatalf("should be standby")
	}

	// Wait for core2 to become active
	start = time.Now()
	for time.Now().Sub(start) < time.Second {
		standby, err = core2.Standby()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !standby {
			break
		}
	}
	if standby {
		t.Fatalf("should not be in standby mode")
	}

	// Read the secret
	req = &logical.Request{
		Operation:   logical.ReadOperation,
		Path:        "secret/foo",
		ClientToken: root,
	}
	resp, err := core2.HandleRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if resp.Data["foo"] != "bar" {
		t.Fatalf("bad: %#v", resp)
	}

	// Check the leader is local
	isLeader, advertise, err = core2.Leader()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isLeader {
		t.Fatalf("should be leader")
	}
	if advertise != "bar" {
		t.Fatalf("Bad advertise: %v", advertise)
	}
}
