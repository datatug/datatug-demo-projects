SELECT
    i.InvoiceId,
    i.InvoiceDate,
    i.BillingCity,
    i.BillingCountry,
    i.Total
FROM Invoice AS i
WHERE i.CustomerId = @CustomerId
ORDER BY i.InvoiceDate DESC
