import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import grpc from 'k6/net/grpc';

const client = new grpc.Client();
client.load(['.'], 'cryptobridge.proto');

function loadData(){
  const file = open('./provider-avail-markets.json');
  const data = JSON.parse(file);
  return {
    'binance': new SharedArray('BinanceMarkets', function () {
      return data['binance'];
    }),
    'kucoin': new SharedArray('KucoinMarkets', function () {
      return data['kucoin'];
    }),
    'huobi': new SharedArray('HuobiMarkets', function () {
      return data['huobi'];
    }),
    'gateio': new SharedArray('GateioMarkets', function () {
      return data['gateio'];
    }),
  }
}

const availableMarketsData = loadData();


function rundom(min, max) {
  return Math.floor(Math.random() * (max - min)) + min;
}

const getProvider = () => {
  const providersToTest = ['kucoin', ];
  return providersToTest[rundom(0, providersToTest.length)];
}

const getMarket = (provider) => {
  const list = availableMarketsData[provider];
  return list[rundom(0, list.length)]
}

export const options = {  
  vus: 3,
  duration: '1m',
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
  const market = getMarket(provider);
  const [baseAsste, quouteAsset] = market.split("/")

  const data = { provider, market: `${baseAsste}_${quouteAsset}`, maxDepth: "2" };
  console.log(`requested for ${provider} ${market}`)
  const response = client.invoke('CryptoBridge.MarketDataService/GetOrderBookSnapshot', data);


  check(response.message, {
    'bids length > 0': (r) => r && r.bids ,
  });

  check(response.message, {
    'source != Unknown': (r) => r.source !== "Unknown",
  })

  console.log(`response for ${provider} ${market}: `, response.message)


  client.close();
  sleep(1);
};