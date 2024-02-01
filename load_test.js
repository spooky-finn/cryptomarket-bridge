import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import grpc from 'k6/net/grpc';

const client = new grpc.Client();
client.load(['.'], 'cryptobridge.proto');

const data = new SharedArray('some data name', function () {
  return JSON.parse(open('./provider-currencies.json'));
});


function rundom(min, max) {
  return Math.floor(Math.random() * (max - min)) + min;
}

const getProvider = () => {
  const providers = [ 'kucoin'];
  return providers[rundom(0, 1)];
}

const getMarket = (provider) => {
  // return data[rundom(0, data.length - 1)]
  const markets = ["btc/usdt", "trx/btc"]
  return markets[rundom(0, markets.length) ]
}

export const options = {  
  vus: 20,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<1000'],
  },
};

export default () => {
    client.connect('localhost:50051', {
      plaintext: true,
      reflect: false,
      timeout: 10000,
    });

  
  const provider = getProvider();
  const market = getMarket();
  const data = { provider, market, maxDepth: "2" };
  console.log(`requested for ${provider} ${market}`)
  const response = client.invoke('CryptoBridge.MarketDataService/GetOrderBookSnapshot', data);


  check(response.message, {
    'bids length > 0': (r) => r && r.bids ,
  });
  console.log(`response for ${provider} ${market}: `, response.message)


  client.close();
  sleep(1);
};