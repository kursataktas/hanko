package shared

import "github.com/teamhanko/hanko/backend/flowpilot"

const (
	StateError                                 flowpilot.StateName = "error"
	StateLoginInit                             flowpilot.StateName = "login_init"
	StateLoginMethodChooser                    flowpilot.StateName = "login_method_chooser"
	StateLoginPasskey                          flowpilot.StateName = "login_passkey"
	StateLoginPassword                         flowpilot.StateName = "login_password"
	StateLoginPasswordRecovery                 flowpilot.StateName = "login_password_recovery"
	StateOnboardingCreatePasskey               flowpilot.StateName = "onboarding_create_passkey"
	StateOnboardingVerifyPasskeyAttestation    flowpilot.StateName = "onboarding_verify_passkey_attestation"
	StatePasscodeConfirmation                  flowpilot.StateName = "passcode_confirmation"
	StatePasswordCreation                      flowpilot.StateName = "password_creation"
	StatePreflight                             flowpilot.StateName = "preflight"
	StateProfileAccountDeleted                 flowpilot.StateName = "account_deleted"
	StateProfileInit                           flowpilot.StateName = "profile_init"
	StateProfileWebauthnCredentialVerification flowpilot.StateName = "webauthn_credential_verification"
	StateRegisterPasskey                       flowpilot.StateName = "register_passkey"
	StateRegistrationInit                      flowpilot.StateName = "registration_init"
	StateRegistrationMethodChooser             flowpilot.StateName = "registration_method_chooser"
	StateSuccess                               flowpilot.StateName = "success"
	StateThirdPartyOAuth                       flowpilot.StateName = "thirdparty_oauth"
)
