describe Chain::AuthorizationGrant::ClientModule do

  describe 'sanitize_x509' do

    example 'arrayification of attributes' do

      expect(
        Chain::AuthorizationGrant::ClientModule.sanitize_x509(
          'subject' => {
            'C' => 'foo',
            'O' => 'foo',
            'OU' => 'foo',
            'L' => 'foo',
            'ST' => 'foo',
            'STREET' => 'foo',
            'POSTALCODE' => 'foo',
            'SERIALNUMBER' => 'foo',
            'CN' => 'foo',
          }
        )
      ).to eq(
        'subject' => {
          'C' => ['foo'],
          'O' => ['foo'],
          'OU' => ['foo'],
          'L' => ['foo'],
          'ST' => ['foo'],
          'STREET' => ['foo'],
          'POSTALCODE' => ['foo'],
          'SERIALNUMBER' => 'foo',
          'CN' => 'foo',
        }
      )

    end

    describe 'error cases' do

      example 'multiple-top-level attributes' do
        expect {
          Chain::AuthorizationGrant::ClientModule.sanitize_x509(
            'subject' => {},
            'foobar' => {},
          )
        }.to raise_error(ArgumentError)
      end

      example 'non-subject top-level attribute' do
        expect {
          Chain::AuthorizationGrant::ClientModule.sanitize_x509(
            'foobar' => {},
          )
        }.to raise_error(ArgumentError)
      end

      example 'bad attribute names' do
        expect {
          Chain::AuthorizationGrant::ClientModule.sanitize_x509(
            'subject' => {
              'C' => 'okay',
              'Foo' => 'invalid',
            },
          )
        }.to raise_error(ArgumentError)
      end

      example 'invalid array attributes' do
        expect {
          Chain::AuthorizationGrant::ClientModule.sanitize_x509(
            'subject' => {
              'CN' => ['invalid'],
            },
          )
        }.to raise_error(ArgumentError)
      end

    end

  end

end
