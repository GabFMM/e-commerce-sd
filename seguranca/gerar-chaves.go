// Rodar UMA VEZ, antes de subir qualquer microsserviço
package seguranca

import (
	"fmt"
	"io"
	"log"
	"os"
)

func copiarArquivo(origem, destino string) error {
	entrada, err := os.Open(origem)
	if err != nil {
		return err
	}
	defer entrada.Close()

	saida, err := os.Create(destino)
	if err != nil {
		return err
	}
	defer saida.Close()

	_, err = io.Copy(saida, entrada)
	return err
}

// nomesServicos lista todos os microsserviços que precisam de chaves.
// Ajuste aqui se adicionar/remover algum serviço.
var nomesServicos = []string{
	"principal",
	"estoque",
	"pagamento",
	"entrega",
	"promocoes",
}

// pastaChavesDoServico monta o caminho da pasta "chaves" de um serviço,
// assumindo que este script roda a partir de "gerar-chaves/" e que
// cada microsserviço mora em "ms-<nome>/".
func pastaChavesDoServico(nomeServico string) string {
	return fmt.Sprintf("ms-%s/chaves", nomeServico)
}

func GerarChaves() {
	// Gera o par de chaves de cada microsserviço, direto dentro
	// da sua própria pasta "chaves/" (assim a privada já nasce no
	// lugar certo e nunca precisa ser copiada para lugar nenhum).
	for _, nome := range nomesServicos {
		pasta := pastaChavesDoServico(nome)
		if err := GerarEGuardarChaves(pasta, nome); err != nil {
			log.Fatalf("falha ao gerar chaves de %q: %v", nome, err)
		}
	}

	// Para cada serviço, copia a
	// chave pública de todos os OUTROS serviços para dentro da sua
	// pasta "chaves/". Nunca copiamos chave privada.
	for _, destino := range nomesServicos {
		pastaDestino := pastaChavesDoServico(destino)

		for _, origem := range nomesServicos {
			if origem == destino {
				continue // não precisa copiar a própria chave pública
			}

			nomeArquivo := fmt.Sprintf("%s_public.pem", origem)
			caminhoOrigem := fmt.Sprintf("%s/%s", pastaChavesDoServico(origem), nomeArquivo)
			caminhoDestino := fmt.Sprintf("%s/%s", pastaDestino, nomeArquivo)

			if err := copiarArquivo(caminhoOrigem, caminhoDestino); err != nil {
				log.Fatalf("falha ao copiar %s para %s: %v", caminhoOrigem, caminhoDestino, err)
			}
		}
	}

	fmt.Println("Todas as chaves foram geradas e distribuídas com sucesso.")
}
