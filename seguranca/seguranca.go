package seguranca

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

type Pacote struct {
	Payload   json.RawMessage `json:"payload"`
	Producer  string          `json:"producer"`
	Signature []byte          `json:"signature"`
}

func salvarChavePublicaPEM(caminho string, pub ed25519.PublicKey) error {

	bytesChave, err := x509.MarshalPKIXPublicKey(pub)

	if err != nil {
		return fmt.Errorf("falha ao serializar chave pública: %w", err)
	}

	arquivo, err := os.Create(caminho)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo %s: %w", caminho, err)
	}
	defer arquivo.Close()

	return pem.Encode(arquivo, &pem.Block{Type: "PUBLIC KEY", Bytes: bytesChave})
}

func salvarChavePrivadaPEM(caminho string, priv ed25519.PrivateKey) error {
	bytesChave, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("falha ao serializar chave privada: %w", err)
	}

	arquivo, err := os.OpenFile(caminho, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo %s: %w", caminho, err)
	}
	defer arquivo.Close()

	return pem.Encode(arquivo, &pem.Block{Type: "PRIVATE KEY", Bytes: bytesChave})
}

func GerarEGuardarChaves(pastaDestino, nomeServico string) error {
	chavePublica, chavePrivada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("Falha ao gerar chaves: %w", err)
	}

	if err := os.MkdirAll(pastaDestino, 0700); err != nil {
		return fmt.Errorf("Falha ao criar pasta %s: %w ", pastaDestino, err)
	}

	caminhoPublica := fmt.Sprintf("%s/%s_public.pem", pastaDestino, nomeServico)
	caminhoPrivada := fmt.Sprintf("%s/%s_private.pem", pastaDestino, nomeServico)

	if err := salvarChavePublicaPEM(caminhoPublica, chavePublica); err != nil {
		return err
	}
	if err := salvarChavePrivadaPEM(caminhoPrivada, chavePrivada); err != nil {
		return err
	}

	fmt.Printf("Chaves de %q geradas em %s\n", nomeServico, pastaDestino)
	return nil
}

func CarregarChavePublica(caminho string) (ed25519.PublicKey, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler %s: %w", caminho, err)
	}

	bloco, _ := pem.Decode(dados)
	if bloco == nil {
		return nil, fmt.Errorf("arquivo %s não contém PEM válido", caminho)
	}

	chaveGenerica, err := x509.ParsePKIXPublicKey(bloco.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear chave pública: %w", err)
	}

	chavePublica, ok := chaveGenerica.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("chave lida não é do tipo Ed25519")
	}

	return chavePublica, nil
}

// CarregarChavePrivada lê a própria chave privada do microsserviço.
func CarregarChavePrivada(caminho string) (ed25519.PrivateKey, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler %s: %w", caminho, err)
	}

	bloco, _ := pem.Decode(dados)
	if bloco == nil {
		return nil, fmt.Errorf("arquivo %s não contém PEM válido", caminho)
	}

	chaveGenerica, err := x509.ParsePKCS8PrivateKey(bloco.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear chave privada: %w", err)
	}

	chavePrivada, ok := chaveGenerica.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("chave lida não é do tipo Ed25519")
	}

	return chavePrivada, nil
}

// CriarPacote serializa o payload, assina e monta o Pacote pronto para ir no Body da mensagem AMQP.
func CriarPacote(payload interface{}, nomeServico string, chavePrivada ed25519.PrivateKey) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar payload: %w", err)
	}

	//gera o hash do conteúdo (payload + identidade do produtor)
	dados := append([]byte(nomeServico+":"), payloadBytes...)
	hash := sha256.Sum256(dados)

	//assina o hash com a chave privada
	assinatura := ed25519.Sign(chavePrivada, hash[:])

	//inclui a assinatura no campo Signature
	pacote := Pacote{
		Payload:   payloadBytes,
		Producer:  nomeServico,
		Signature: assinatura,
	}

	return json.Marshal(pacote)
}

// AbrirPacote desserializa o Pacote e só retorna o payload se a
// assinatura for válida; caso contrário retorna erro (evento deve ser descartado pelo chamador).
func AbrirPacote(bodyMensagem []byte, chavesPublicas map[string]ed25519.PublicKey) (json.RawMessage, error) {
	var Pacote Pacote
	if err := json.Unmarshal(bodyMensagem, &Pacote); err != nil {
		return nil, fmt.Errorf("falha ao desserializar Pacote: %w", err)
	}

	chavePublica, existe := chavesPublicas[Pacote.Producer]
	if !existe {
		return nil, fmt.Errorf("chave pública desconhecida para produtor %q", Pacote.Producer)
	}

	if !ed25519.Verify(chavePublica, Pacote.Payload, Pacote.Signature) {
		return nil, fmt.Errorf("assinatura inválida — evento de %q descartado", Pacote.Producer)
	}

	return Pacote.Payload, nil
}

// CarregarTodasChavesPublicas varre uma pasta e monta o mapa
// nomeServico -> chave pública, a partir de arquivos "<nome>_public.pem". Chamar uma vez na inicialização do microsserviço.
func CarregarTodasChavesPublicas(pasta string) (map[string]ed25519.PublicKey, error) {
	entradas, err := os.ReadDir(pasta)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler pasta %s: %w", pasta, err)
	}

	const sufixo = "_public.pem"
	chaves := make(map[string]ed25519.PublicKey)

	for _, entrada := range entradas {
		nomeArquivo := entrada.Name()
		if entrada.IsDir() || len(nomeArquivo) <= len(sufixo) || nomeArquivo[len(nomeArquivo)-len(sufixo):] != sufixo {
			continue
		}

		nomeServico := nomeArquivo[:len(nomeArquivo)-len(sufixo)]
		chavePublica, err := CarregarChavePublica(fmt.Sprintf("%s/%s", pasta, nomeArquivo))
		if err != nil {
			return nil, fmt.Errorf("falha ao carregar chave de %s: %w", nomeArquivo, err)
		}

		chaves[nomeServico] = chavePublica
	}
	return chaves, nil
}
