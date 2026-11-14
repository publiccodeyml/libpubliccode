package validators

import (
	"errors"
	"testing"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

func TestNewReturnsNonNil(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}
}

func TestRegisterLocalErrorMessages(t *testing.T) {
	v := New()
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(v, trans)

	err := RegisterLocalErrorMessages(v, trans)
	if err != nil {
		t.Fatalf("RegisterLocalErrorMessages returned error: %v", err)
	}
}

func TestNewTagNameFuncWithDashTag(t *testing.T) {
	v := New()

	// A struct with yaml:"-" should have the field excluded from validation.
	type S struct {
		Name   string `validate:"required" yaml:"name"`
		Hidden string `yaml:"-"`
	}

	// Validate a struct with yaml:"-" to trigger the name=="-" path.
	err := v.Struct(S{Name: "test", Hidden: ""})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegisterLocalErrorMessagesOrgURIAlreadyRegistered(t *testing.T) {
	// Pre-register "organisation_uri" so the customRegisFunc for that tag
	// fails on ut.Add, covering the error-return paths in RegisterLocalErrorMessages.
	v := New()
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(v, trans)

	_ = trans.Add("organisation_uri", "already registered", false)

	err := RegisterLocalErrorMessages(v, trans)
	if err == nil {
		t.Error("expected error when organisation_uri is already registered")
	}
}

func TestRegisterLocalErrorMessagesOrgURIInvalidIPAAlreadyRegistered(t *testing.T) {
	// Pre-register only organisation_uri_invalid_italian_pa so the first ut.Add
	// in customRegisFunc succeeds but the second fails, covering the second
	// error-return path.
	v := New()
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(v, trans)

	_ = trans.Add("organisation_uri_invalid_italian_pa", "already registered", false)

	err := RegisterLocalErrorMessages(v, trans)
	if err == nil {
		t.Error("expected error when organisation_uri_invalid_italian_pa is already registered")
	}
}

func TestTranslateFunc(t *testing.T) {
	v := New()
	type S struct {
		Lang string `validate:"bcp47_strict_language_tag"`
	}

	err := v.Struct(S{Lang: "not-a-valid-lang-123456789"})
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) || len(validationErrors) != 1 {
		t.Fatalf("expected one validation error, got %v", err)
	}
	fe := validationErrors[0]

	for _, tc := range []struct {
		name        string
		translation string
		want        string
	}{
		{name: "missing key", want: fe.Error()},
		{name: "registered key", translation: "{0} is not a language", want: "Lang is not a language"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enLocale := en.New()
			uni := ut.New(enLocale, enLocale)
			trans, _ := uni.GetTranslator("en")
			if tc.translation != "" {
				if err := trans.Add(fe.Tag(), tc.translation, false); err != nil {
					t.Fatalf("registering translation: %v", err)
				}
			}

			if got := translateFunc(trans, fe); got != tc.want {
				t.Errorf("translateFunc() = %q, want %q", got, tc.want)
			}
		})
	}
}
