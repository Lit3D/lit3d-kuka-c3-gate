<template id="infoTemplate">
	<h1>Bot&emsp;${this.id}:&emsp;&emsp;${this.Name}&emsp;&emsp;[${this.Address}]</h1>
	<p>
		<span><strong>AxisPath:</strong>&emsp;&emsp;${this.OSCRequestAxis || "Ø"}</span>
		<span><strong>CoordsPath:</strong>&emsp;&emsp;${this.OSCRequestCoords || "Ø"}</span>
		<span><strong>PositionPath:</strong>&emsp;&emsp;${this.OSCRequestPosition || "Ø"}</span>
	</p>
	<p>
		<span><strong>ResponseAddress:</strong>&emsp;&emsp;${this.OSCResponseAddress || "Ø"}</span>
		<span><strong>ResponseAxes:</strong>&emsp;&emsp;${this.OSCResponseAxes || "Ø"}</span>
		<span><strong>ResponseCoords:</strong>&emsp;&emsp;${this.OSCResponseCoords || "Ø"}</span>
		<span><strong>ResponsePosition:</strong>&emsp;&emsp;${this.OSCResponsePosition || "Ø"}</span>
	</p>
	<p>
		<span><strong>ProxyType:</strong>&emsp;&emsp;${this.PROXY_TYPE || "Ø"}</span>
		<span><strong>ProxyVersion:</strong>&emsp;&emsp;${this.PROXY_VERSION || "Ø"}</span>
	</p>
	<p>
		<span><strong>ProxyHost:</strong>&emsp;&emsp;${this.PROXY_HOSTNAME || "Ø"}</span>
		<span><strong>ProxyAddress:</strong>&emsp;&emsp;${this.PROXY_ADDRESS || "Ø"}</span>
		<span><strong>ProxyPort:</strong>&emsp;&emsp;${this.PROXY_PORT || "Ø"}</span>
	</p>
	<div class="two-columns">
		<p><strong>AXIS_ACT:</strong><span>${this.AXIS_ACT}</span></p>
		<p><strong>POS_ACT:</strong><span>${this.POS_ACT}</span></p>
		<p><strong>OFFSET:</strong><span>${this.OFFSET}</span></p>
		<p><strong>POSITION:</strong><span>${this.POSITION}</span></p>
	</div>
	<p>
		<span><strong>COM_ACTION:</strong>&emsp;&emsp;${this.COM_ACTION}</span>
		<span><strong>COM_ROUNDM:</strong>&emsp;&emsp;${this.COM_ROUNDM}</span>
		<span><strong>isMovement:</strong>&emsp;&emsp;${this.IsMovement}</span>
		<span><strong>tagId:</strong>&emsp;&emsp;${this.TagId}</span>
	</p>
</template>

<section id="infoSection" class="bot-info">
	
</section>

<section id="moveSection" class="bot-move">
	
</section>