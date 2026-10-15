// Google login (docs/04 §9): the Google account email becomes the identity email. Google has
// verified it already, so it counts as verified; an unverified Google email is not accepted.
local claims = std.extVar('claims');

if !std.objectHas(claims, 'email') || !std.objectHas(claims, 'email_verified') || !claims.email_verified then
  error 'Google account email is missing or not verified'
else {
  identity: {
    traits: {
      email: claims.email,
    },
    verified_addresses: [
      { via: 'email', value: claims.email },
    ],
  },
}
