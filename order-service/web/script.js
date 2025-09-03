function fetchOrder() {
    const id = document.getElementById('orderId').value.trim();
    if (!id) {
        document.getElementById('result').textContent = 'Введите order_uid';
        return;
    }

    fetch('/order/' + encodeURIComponent(id))
        .then(resp => {
            if (!resp.ok) throw new Error('Заказ не найден');
            return resp.json();
        })
        .then(data => {
            const result = document.getElementById('result');
            result.innerHTML = renderOrder(data);
        })
        .catch(e => {
            const result = document.getElementById('result');
            result.textContent = e.message;
        });
}

// Функция для форматирования и вывода данных заказа
function renderOrder(order) {
    function formatDate(ts) {
        const d = new Date(ts * 1000);
        return d.toLocaleString();
    }

    function formatISODate(str) {
        const d = new Date(str);
        return d.toLocaleString();
    }

    return `
    <h3>Заказ: ${order.order_uid}</h3>
    <strong>Трек номер:</strong> ${order.track_number}<br>
    <strong>Дата создания:</strong> ${formatISODate(order.date_created)}

    <h4>Доставка</h4>
    Имя: ${order.delivery.name}<br>
    Телефон: ${order.delivery.phone}<br>
    Адрес: ${order.delivery.address}, ${order.delivery.city}, ${order.delivery.region}, ${order.delivery.zip}<br>
    Email: ${order.delivery.email}

    <h4>Оплата</h4>
    Транзакция: ${order.payment.transaction} 
    Провайдер: ${order.payment.provider}<br>
    Сумма: ${order.payment.amount} ${order.payment.currency}<br>
    Стоимость доставки: ${order.payment.delivery_cost} ${order.payment.currency}<br>
    Дата оплаты: ${formatDate(order.payment.payment_dt)}<br>
    Банк: ${order.payment.bank}

    <h4>Товары (${order.items.length})</h4>
    <ul>
        ${order.items.map(item => `
            <li>
                <strong>${item.name}</strong> (бренд: ${item.brand})<br>
                Цена: ${item.price} ${order.payment.currency}, скидка: ${item.sale}%, итог: ${item.total_price}<br>
                Размер: ${item.size}
            </li>
        `).join('')}
    </ul>

    <em>Служба доставки: ${order.delivery_service}</em>
`;

}
