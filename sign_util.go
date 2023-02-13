package aliyundrive

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"

	"github.com/dustinxie/ecc"
)

var (
	_fn, _ = new(big.Int).SetString("115792089237316195423570985008687907852837564279074904382605163141518161494337", 10)
	_i     = big.NewInt(0)
	_o     = big.NewInt(1)
)

type _V struct {
	v []byte
	k []byte
}

type _p struct {
	r *big.Int
	s *big.Int
}

func (v *_p) hasHighS() bool {
	t := new(big.Int).Rsh(_fn, 1)
	return v.s.Cmp(t) == 1
}

func (v *_p) normalizeS() *_p {
	if v.hasHighS() {
		return &_p{r: v.r, s: new(big.Int).Sub(_fn, v.s)}
	}
	return v
}

func (v *_p) toCompactRawBytes() []byte {
	r := v.r.Bytes()
	s := v.s.Bytes()
	ret := make([]byte, 64)
	copy(ret[32-len(r):], r)
	copy(ret[64-len(s):], s)
	return ret
}

func _newV() *_V {
	v := &_V{
		v: make([]byte, 32),
		k: make([]byte, 32),
	}
	for i := range v.v {
		v.v[i] = 1
	}
	return v
}

func (v *_V) reseed(t []byte) {
	v.k = v.hmac(v.v, []byte{0}, t)
	v.v = v.hmac(v.v)
	if len(t) > 0 {
		v.k = v.hmac(v.v, []byte{1}, t)
		v.v = v.hmac(v.v)
	}
}

func (v *_V) generate() []byte {
	v.v = v.hmac(v.v)
	return v.v
}

func (v *_V) hmac(t ...[]byte) []byte {
	hasher := hmac.New(sha256.New, v.k)
	for _, sub := range t {
		hasher.Write(sub)
	}
	return hasher.Sum(nil)
}

func _P(t *big.Int, e *big.Int) *big.Int {
	r := new(big.Int).Mod(t, e)
	if r.Cmp(_i) == -1 {
		return new(big.Int).Add(e, r)
	}
	return r
}

func _Q(t []byte, e []byte) (seed []byte, m *big.Int, d *big.Int) {
	le := _L(e)
	jt := _j(t)
	seed = make([]byte, len(le)+len(jt))
	copy(seed, le)
	copy(seed[len(le):], jt)
	d = new(big.Int).SetBytes(e)
	m = new(big.Int).SetBytes(t)
	return
}

func _L(t []byte) []byte {
	ret := make([]byte, 32)
	copy(ret[32-len(t):], t)
	return ret
}

func _j(t []byte) []byte {
	e := new(big.Int).SetBytes(t)
	r := _P(e, _fn)
	if r.Cmp(_i) == -1 {
		return _L(e.Bytes())
	}
	return _L(r.Bytes())
}

func _K(t *big.Int) bool {
	return _i.Cmp(t) == -1 && t.Cmp(_fn) == -1
}

func _D(t []byte, e *big.Int, r *big.Int) *_p {
	n := new(big.Int).SetBytes(t)
	if !_K(n) {
		return nil
	}

	ax, _ := ecc.P256k1().ScalarBaseMult(t)
	c := _P(ax, _fn)
	if c.Cmp(_i) == 0 {
		return nil
	}

	p1 := new(big.Int).Add(e, (new(big.Int).Mul(r, c)))
	p2 := new(big.Int).Mul(_dollar(n, _fn), _P(p1, _fn))
	u := _P(p2, _fn)

	if u.Cmp(_i) == 0 {
		return nil
	}

	return &_p{r: c, s: u}
}

func _dollar(t *big.Int, e *big.Int) *big.Int {
	r := _P(t, e)
	n := e
	s := _i
	a := _o
	c := _o
	u := _i
	for {
		if r.Cmp(_i) == 0 {
			break
		}

		t := new(big.Int).Div(n, r)
		e := new(big.Int).Mod(n, r)
		i := new(big.Int).Sub(s, new(big.Int).Mul(c, t))
		o := new(big.Int).Sub(a, new(big.Int).Mul(u, t))

		n = r
		r = e
		s = c
		a = u
		c = i
		u = o
	}
	return _P(s, e)
}

func _Y(r *_p) []byte {
	if r.hasHighS() {
		r = r.normalizeS()
	}
	return r.toCompactRawBytes()
}

func Sign(hashSum []byte, privateKey []byte) []byte {
	n, i, o := _Q(hashSum, privateKey)
	a := _newV()
	a.reseed(n)
	var s *_p
	for {
		s = _D(a.generate(), i, o)
		if s != nil {
			break
		}
		a.reseed([]byte{})
	}
	return _Y(s)
}
