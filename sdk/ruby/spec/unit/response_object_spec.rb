class Foo < Chain::ResponseObject
  attrib :time, rfc3339_time: true
end

class Bar < Chain::ResponseObject
  attrib :time, rfc3339_time: true
  attrib(:foos) { |raw| raw.map { |item| Foo.new(item) } }
end

describe Chain::ResponseObject do

  describe 'translation and detranslation' do

    it 'handles nested time translation' do
      # DateTime's to_rfc3339 method uses numeric timezones, so this is what
      # we'll match against here.
      t1 = '2017-01-01T00:00:00+00:00'
      t2 = '2018-01-01T00:00:00+00:00'
      t3 = '2019-01-01T00:00:00+00:00'

      raw = {time: t1, foos: [{time: t2}, {time: t3}]}
      b = Bar.new(raw)

      expect(b.time).to eql(Time.parse(t1))
      expect(b.foos[0].time).to eql(Time.parse(t2))
      expect(b.foos[1].time).to eql(Time.parse(t3))
      expect(b.to_json).to eql(raw.to_json)
    end

  end

end
