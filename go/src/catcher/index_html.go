package catcher

const IndexHTML = `<html>
	<head>
		<title>Catcher</title>
		<script>
			window._webSocket = null
			let wsMessageEl = null
			function init(ev){
				wsMessageEl = document.querySelector('#ws-message')
				if (!wsMessageEl) {
					console.error('Could not locate the message element')
					return
				}

				window._webSocket = new WebSocket('ws://' + document.location.host + '/echo')
				window._webSocket.onconnect = () => { console.log('connected') }
				window._webSocket.onmessage = (message) => {
					const date = new Date()
					date.setTime(message.data)
					const dateStr = (date.getHours() + '').padStart(2, '0') + ':' + (date.getMinutes() + '').padStart(2, '0') + ':' + (date.getSeconds() + '').padStart(2, '0')
					wsMessageEl.innerText = 'WebSocket Echo: ' + dateStr
				}

				setInterval(() => {
					window._webSocket.send(Date.now())
				}, 1000)
			}
			document.addEventListener('DOMContentLoaded', init)
		</script>
	</head>
	<body>
		<h1>This is Catcher</h1>
		<div id="ws-message">Connecting to WebSocket...</div>
	</body>
</html>`
