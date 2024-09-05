package shared

import "github.com/teamhanko/hanko/backend/flowpilot"

const (
	ActionAccountDelete                          flowpilot.ActionName = "account_delete"
	ActionBack                                   flowpilot.ActionName = "back"
	ActionContinueToLoginOTP                     flowpilot.ActionName = "continue_to_login_otp"
	ActionContinueToLoginSecurityKey             flowpilot.ActionName = "continue_to_login_security_key"
	ActionContinueToOTPSecretCreation            flowpilot.ActionName = "continue_to_otp_secret_creation"
	ActionContinueToPasscodeConfirmation         flowpilot.ActionName = "continue_to_passcode_confirmation"
	ActionContinueToPasscodeConfirmationRecovery flowpilot.ActionName = "continue_to_passcode_confirmation_recovery"
	ActionContinueToPasskeyRegistration          flowpilot.ActionName = "continue_to_passkey_registration"
	ActionContinueToPasswordLogin                flowpilot.ActionName = "continue_to_password_login"
	ActionContinueToPasswordRegistration         flowpilot.ActionName = "continue_to_password_registration"
	ActionContinueToSecurityKeyCreation          flowpilot.ActionName = "continue_to_security_key_creation"
	ActionContinueWithLoginIdentifier            flowpilot.ActionName = "continue_with_login_identifier"
	ActionEmailAddressSet                        flowpilot.ActionName = "email_address_set"
	ActionEmailCreate                            flowpilot.ActionName = "email_create"
	ActionEmailDelete                            flowpilot.ActionName = "email_delete"
	ActionEmailSetPrimary                        flowpilot.ActionName = "email_set_primary"
	ActionEmailVerify                            flowpilot.ActionName = "email_verify"
	ActionExchangeToken                          flowpilot.ActionName = "exchange_token"
	ActionOTPCodeValidate                        flowpilot.ActionName = "otp_code_validate"
	ActionOTPCodeVerify                          flowpilot.ActionName = "otp_code_verify"
	ActionPasswordCreate                         flowpilot.ActionName = "password_create"
	ActionPasswordDelete                         flowpilot.ActionName = "password_delete"
	ActionPasswordLogin                          flowpilot.ActionName = "password_login"
	ActionPasswordRecovery                       flowpilot.ActionName = "password_recovery"
	ActionPasswordUpdate                         flowpilot.ActionName = "password_update"
	ActionRegisterClientCapabilities             flowpilot.ActionName = "register_client_capabilities"
	ActionRegisterLoginIdentifier                flowpilot.ActionName = "register_login_identifier"
	ActionRegisterPassword                       flowpilot.ActionName = "register_password"
	ActionResendPasscode                         flowpilot.ActionName = "resend_passcode"
	ActionSkip                                   flowpilot.ActionName = "skip"
	ActionThirdPartyOAuth                        flowpilot.ActionName = "thirdparty_oauth"
	ActionUsernameCreate                         flowpilot.ActionName = "username_create"
	ActionUsernameDelete                         flowpilot.ActionName = "username_delete"
	ActionUsernameUpdate                         flowpilot.ActionName = "username_update"
	ActionVerifyPasscode                         flowpilot.ActionName = "verify_passcode"
	ActionWebauthnCredentialCreate               flowpilot.ActionName = "webauthn_credential_create"
	ActionWebauthnCredentialDelete               flowpilot.ActionName = "webauthn_credential_delete"
	ActionWebauthnCredentialRename               flowpilot.ActionName = "webauthn_credential_rename"
	ActionWebauthnGenerateCreationOptions        flowpilot.ActionName = "webauthn_generate_creation_options"
	ActionWebauthnGenerateRequestOptions         flowpilot.ActionName = "webauthn_generate_request_options"
	ActionWebauthnVerifyAssertionResponse        flowpilot.ActionName = "webauthn_verify_assertion_response"
	ActionWebauthnVerifyAttestationResponse      flowpilot.ActionName = "webauthn_verify_attestation_response"
	ActionSessionDelete                          flowpilot.ActionName = "session_delete"
)
