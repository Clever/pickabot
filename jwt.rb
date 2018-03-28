require 'openssl'
require 'jwt'  # https://rubygems.org/gems/jwt

# Private key contents
private_pem = File.read("pickabot.private-key.pem")
private_key = OpenSSL::PKey::RSA.new(private_pem)

# Generate the JWT
payload = {
  # issued at time
  iat: Time.now.to_i,
  # JWT expiration time (10 minute maximum)
  exp: Time.now.to_i + (10 * 60),
  # JWT expiration time (1 year maximum)
  #exp: Time.now.to_i + (365 * 24 * 60 * 60),
  # GitHub App's identifier
  iss: "9583",  
  # Clever's installation ID == 96158
}

jwt = JWT.encode(payload, private_key, "RS256")
puts jwt

# Add a JWT lib to pickabot
